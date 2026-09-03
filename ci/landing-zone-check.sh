#!/usr/bin/env bash
# "Cloud-agnostic" is a claim, and an untested claim is a wish.
#
# The landing zone is the least portable layer in any platform: org hierarchies,
# IAM models and billing structures genuinely differ between providers, and a
# module that pretends otherwise produces a lowest-common-denominator abstraction
# nobody can use. What IS portable is the interface — identical inputs, identical
# outputs — with a separate implementation behind each.
#
# This asserts that parity. It exists to catch drift: the day someone adds a
# GCP-only variable to the shared interface, the portability claim quietly dies
# unless something fails.
set -euo pipefail

ROOT=${ROOT:-examples/landing-zone/modules}
REFERENCE=${REFERENCE:-gcp}
fails=0

modules=$(find "$ROOT" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort)
echo "modules: $(echo "$modules" | tr '\n' ' ')"

names() { # names <file> <keyword>
  grep -oE "^${2} \"[a-z0-9_]+\"" "$1" 2>/dev/null | sed -E "s/^${2} \"(.*)\"/\1/" | sort
}

echo
echo "terraform fmt"
if terraform fmt -check -recursive "$ROOT" >/dev/null 2>&1; then
  echo "  ok    formatted"
else
  echo "  FAIL  run: terraform fmt -recursive $ROOT"; fails=$((fails+1))
fi

echo "terraform validate"
for m in $modules; do
  if (cd "$ROOT/$m" && terraform init -backend=false >/dev/null 2>&1 \
        && terraform validate >/dev/null 2>&1); then
    echo "  ok    $m"
  else
    echo "  FAIL  $m does not validate"; fails=$((fails+1))
  fi
done

echo "interface parity against '$REFERENCE'"
for kind in variable output; do
  file=$([ "$kind" = variable ] && echo variables.tf || echo outputs.tf)
  ref=$(names "$ROOT/$REFERENCE/$file" "$kind")
  [ -z "$ref" ] && { echo "  FAIL  reference module declares no ${kind}s"; fails=$((fails+1)); continue; }

  for m in $modules; do
    [ "$m" = "$REFERENCE" ] && continue
    got=$(names "$ROOT/$m/$file" "$kind")

    missing=$(comm -23 <(echo "$ref") <(echo "$got") | tr '\n' ' ' | sed 's/ *$//')
    extra=$(comm -13 <(echo "$ref") <(echo "$got") | tr '\n' ' ' | sed 's/ *$//')

    if [ -z "$missing" ] && [ -z "$extra" ]; then
      echo "  ok    $m ${kind}s match ($(echo "$got" | wc -l | tr -d ' ') declared)"
    else
      [ -n "$missing" ] && { echo "  FAIL  $m is missing ${kind}s: $missing"; fails=$((fails+1)); }
      # An extra input in one provider is the failure mode this whole script
      # exists to catch: it means callers can no longer be written once.
      [ -n "$extra" ] && { echo "  FAIL  $m declares ${kind}s the interface does not: $extra"; fails=$((fails+1)); }
    fi
  done
done

echo "no provider names leak into the interface's shape"
# Descriptions are exempt on purpose: an opaque field like parent_scope is only
# usable if its docs say it means a folder on GCP, an OU on AWS and a management
# group on Azure. What must stay neutral is the *shape* — names, types, defaults,
# validation — because that is what callers are written against.
strip_docs() {
  awk '
    /<<-?EOT/       { inheredoc = 1; next }
    inheredoc && /^[[:space:]]*EOT[[:space:]]*$/ { inheredoc = 0; next }
    inheredoc       { next }
    /description[[:space:]]*=/ { next }
    { print }
  ' "$1"
}
leaked=0
for m in $modules; do
  for file in variables.tf outputs.tf; do
    if hit=$(strip_docs "$ROOT/$m/$file" | grep -inE '\b(gcloud|gke|eks|aks|azurerm|arn:aws)' | head -3); then
      echo "  FAIL  $m/$file has a provider name in its shape:"; echo "$hit" | sed 's/^/          /'
      fails=$((fails+1)); leaked=1
    fi
  done
done
[ "$leaked" -eq 0 ] && echo "  ok    names, types and defaults are provider-neutral"

if [ "$fails" -ne 0 ]; then
  echo; echo "landing zone check: $fails failure(s)"; exit 1
fi
echo; echo "landing zone check: interface is identical across $(echo "$modules" | wc -l | tr -d ' ') providers"
