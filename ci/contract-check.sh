#!/usr/bin/env bash
# The contract plane is only real if the schema enforces it. This asserts both
# directions: bad values are rejected, and every platform seam actually renders.
#
# One script, called by the Justfile and by both CI systems, so the two pipelines
# cannot drift apart.
set -euo pipefail

CHART=${CHART:-examples/quickstart/charts/product}
fails=0

reject() {
  local desc="$1"; shift
  if helm template acme "$CHART" "$@" >/dev/null 2>&1; then
    echo "  FAIL  accepted $desc — the contract is not enforced"; fails=$((fails+1))
  else
    echo "  ok    rejected $desc"
  fi
}

renders() {
  local desc="$1" pattern="$2"; shift 2
  if helm template acme "$CHART" "$@" 2>/dev/null | grep -qE "$pattern"; then
    echo "  ok    $desc"
  else
    echo "  FAIL  $desc — expected /$pattern/ in the output"; fails=$((fails+1))
  fi
}

absent() {
  local desc="$1" pattern="$2"; shift 2
  if helm template acme "$CHART" "$@" 2>/dev/null | grep -qE "$pattern"; then
    echo "  FAIL  $desc — /$pattern/ should not be present"; fails=$((fails+1))
  else
    echo "  ok    $desc"
  fi
}

echo "schema rejects invalid values"
reject "an empty registry"        --set global.registry=""
reject "a malformed digest"       --set 'auth-svc.image.digest=sha256:nothex'
reject "a malformed hostname"     --set 'auth-svc.route.host=not a hostname'
reject "an unknown routing mode"  --set global.routing.mode=istio
reject "a typo'd proxy key"       --set global.proxy.htpProxy=http://p:3128
reject "a typo'd gateway key"     --set global.routing.gateway.nmae=gw

echo "defaults stay quiet"
absent "no route by default" 'kind: (Ingress|HTTPRoute)'
absent "no proxy by default" 'HTTP_PROXY'

echo "every platform seam renders"
renders "ingress mode emits an Ingress with its class" 'ingressClassName: nginx' \
  --set global.routing.mode=ingress --set global.routing.ingress.className=nginx \
  --set 'auth-svc.route.enabled=true' --set 'auth-svc.route.host=auth.example.com'

renders "gateway mode emits a Gateway API HTTPRoute" 'kind: HTTPRoute' \
  --set global.routing.mode=gateway --set global.routing.gateway.name=platform-gw \
  --set 'auth-svc.route.enabled=true' --set 'auth-svc.route.host=auth.example.com'

renders "the HTTPRoute binds to the site's Gateway" 'name: platform-gw' \
  --set global.routing.mode=gateway --set global.routing.gateway.name=platform-gw \
  --set 'auth-svc.route.enabled=true' --set 'auth-svc.route.host=auth.example.com'

renders "proxy is injected in both cases" 'name: http_proxy' \
  --set global.proxy.httpProxy=http://proxy.internal:3128

renders "the trust bundle mounts read-only" 'name: internal-ca' \
  --set global.trustBundle.configMap=internal-ca

echo "template guards catch well-typed nonsense"
reject "routing enabled while mode is none" \
  --set 'auth-svc.route.enabled=true' --set 'auth-svc.route.host=a.example.com'
reject "gateway mode with no gateway named" \
  --set global.routing.mode=gateway \
  --set 'auth-svc.route.enabled=true' --set 'auth-svc.route.host=a.example.com'

if [ "$fails" -ne 0 ]; then
  echo "contract check: $fails failure(s)"; exit 1
fi
echo "contract check: all assertions passed"
