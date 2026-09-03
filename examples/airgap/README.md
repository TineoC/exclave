# Air-gapped delivery

Plane 3 on its own: build a package on the connected side, verify it on arrival, deploy it with
no egress.

Needs [`zarf`](https://zarf.dev/) and [`cosign`](https://docs.sigstore.dev/), plus the registry
and cluster that `../quickstart/demo.sh` creates.

```bash
# --- connected side ---------------------------------------------------------
zarf package create . --confirm
cosign sign-blob --yes zarf-package-acme-platform-amd64-3.2.0.tar.zst \
  --output-signature zarf-package.sig

# --- the boundary -----------------------------------------------------------
# Two files cross on approved media: the archive and its signature.

# --- disconnected side ------------------------------------------------------
cosign verify-blob zarf-package-acme-platform-amd64-3.2.0.tar.zst \
  --signature zarf-package.sig --certificate-identity-regexp '.*' \
  --certificate-oidc-issuer-regexp '.*'
zarf package deploy zarf-package-acme-platform-amd64-3.2.0.tar.zst --confirm
```

## What makes this work

**Registry indirection.** `global.registry` is `###ZARF_REGISTRY###` in `site-values.yaml`.
Zarf seeds a registry inside the enclave and substitutes its address at deploy time. Without
that single indirection point in the library chart, every image reference would need patching
by hand on arrival.

**Verify on arrival, not trust on origin.** The signature is checked on the receiving side.
Where the archive came from is not evidence; what it hashes to is.

**No egress.** Images, chart and dependencies are all inside the archive. Nothing resolves a
DNS name that does not exist in the enclave.

> The `--certificate-identity-regexp '.*'` above accepts any signer, which is fine for a demo
> and wrong for production. Pin it to your actual signing identity.
