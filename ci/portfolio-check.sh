#!/usr/bin/env bash
# The portfolio controls are the ones whose failure is silent.
#
# A broken chart fails loudly at deploy time. A broken classification ceiling
# just quietly ingests a descriptor the low side should never hold, and a broken
# redaction produces output that still *looks* redacted. Both are only caught by
# asserting them, so this runs everywhere contract-check.sh does.
set -euo pipefail

C=${C:-examples/contracts/catalog}
F=${F:-examples/contracts/fleet/contracts}
EX="go run ./cmd/exclave"
fails=0

ok()   { echo "  ok    $1"; }
bad()  { echo "  FAIL  $1"; fails=$((fails+1)); }

echo "classification ceiling"
if $EX validate -catalog "$C" -fleet "$F" -max-classification il2 >/dev/null 2>&1; then
  bad "an il2 ceiling accepted an il5/il6 descriptor — the low side can ingest what it must not hold"
else
  ok "il2 ceiling refuses over-classified descriptors"
fi
if $EX validate -catalog "$C" -fleet "$F" -max-classification il6 >/dev/null 2>&1; then
  ok "il6 ceiling loads the whole fleet"
else
  bad "il6 ceiling rejected a descriptor it should allow"
fi
if $EX validate -catalog "$C" -fleet "$F" -max-classification il3 >/dev/null 2>&1; then
  bad "il3 was accepted as a ceiling; it was consolidated into il4 and is not a level"
else
  ok "il3 is rejected as a ceiling"
fi

echo "catalog manifest"
m1=$(mktemp); m2=$(mktemp); trap 'rm -f "$m1" "$m2"' EXIT
$EX manifest -catalog "$C" > "$m1"
$EX manifest -catalog "$C" > "$m2"
if cmp -s "$m1" "$m2"; then
  ok "manifest is byte-identical across runs (a varying manifest cannot be signed)"
else
  bad "manifest is not deterministic"
fi
if $EX verify -catalog "$C" -manifest "$m1" >/dev/null 2>&1; then
  ok "verify passes on an unmodified catalog"
else
  bad "verify failed on an unmodified catalog"
fi

# The scenario the manifest exists for: a constraint or a CVE count edited down.
victim="$C/baseline/4.3.0/release.yaml"
cp "$victim" "$victim.orig"
sed 's/criticalCves: 0/criticalCves: 99/' "$victim.orig" > "$victim"
if $EX verify -catalog "$C" -manifest "$m1" >/dev/null 2>&1; then
  bad "verify accepted a catalog whose CVE count was edited"
else
  ok "verify catches an edited release"
fi
mv "$victim.orig" "$victim"

echo "redaction"
if EXCLAVE_REDACTION_SALT= $EX redact -catalog "$C" -fleet "$F" >/dev/null 2>&1; then
  bad "redaction ran without a salt — unsalted digests over guessable names are reversible"
else
  ok "redaction refuses to run without a salt"
fi

out=$(EXCLAVE_REDACTION_SALT=ci-fixture $EX redact -catalog "$C" -fleet "$F" --format json)
# Every identifier that appears in the fixture descriptors must be absent.
if echo "$out" | grep -iqE 'navy|usaf|army|dla|corp|platform-lab|\.mil|\.smil|folders/|billing|maintenance|il[0-9]'; then
  bad "redacted output contains an identifier:"
  echo "$out" | grep -inE 'navy|usaf|army|dla|corp|platform-lab|\.mil|\.smil|folders/|billing|maintenance|il[0-9]' | head -3 | sed 's/^/          /'
else
  ok "redacted output carries no name, host, folder, billing, window or level"
fi

a=$(EXCLAVE_REDACTION_SALT=s1 $EX redact -catalog "$C" -fleet "$F" | awk 'NR==2{print $1}')
b=$(EXCLAVE_REDACTION_SALT=s1 $EX redact -catalog "$C" -fleet "$F" | awk 'NR==2{print $1}')
c=$(EXCLAVE_REDACTION_SALT=s2 $EX redact -catalog "$C" -fleet "$F" | awk 'NR==2{print $1}')
[ "$a" = "$b" ] && ok "site ids are stable, so a site can be tracked without being named" \
                || bad "site ids are unstable under the same salt"
[ "$a" != "$c" ] && ok "site ids change with the salt, so rotation works" \
                 || bad "site ids did not change when the salt did"

echo "machine-readable output"
if $EX plan -catalog "$C" -fleet "$F" --format json | python3 -m json.tool >/dev/null 2>&1; then
  ok "plan --format json is valid JSON"
else
  bad "plan --format json did not produce valid JSON"
fi

if [ "$fails" -ne 0 ]; then
  echo; echo "portfolio check: $fails failure(s)"; exit 1
fi
echo; echo "portfolio check: all assertions passed"
