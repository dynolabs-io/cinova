#!/usr/bin/env bash
# Bootstrap the App Store Connect + iOS distribution secrets needed by
# .github/workflows/ios.yml on a fresh dynolabs-io/cinova clone.
#
# The 6 shared secrets are re-used verbatim across all Dynolabs apps
# (vcard, cinova, iogrid, …) — they belong to the org's single Apple
# Developer Team. The 7th, IOS_DIST_PROVISION, is per-app because
# provisioning profiles are bound to a specific bundle ID.
#
# Pre-req: `gh auth login` as a user with admin on dynolabs-io/cinova.
# Pre-req: vault access (1Password / lastpass / wherever the cert blobs live).
#
# Run interactively — script prompts for each value once, hides input.
set -euo pipefail

REPO="${REPO:-dynolabs-io/cinova}"

echo "Bootstrapping secrets for ${REPO}"
echo
echo "The 6 shared secrets are the same as dynolabs-io/vcard. Paste each"
echo "value exactly as stored there (verify with: gh secret list -R dynolabs-io/vcard)."
echo

prompt_secret() {
  local name="$1"
  local hint="$2"
  echo "▶ ${name}"
  echo "  hint: ${hint}"
  read -r -s -p "  paste value (input hidden): " val
  echo
  if [ -z "$val" ]; then
    echo "  (empty — skipping ${name})"
    return
  fi
  echo "$val" | gh secret set "$name" -R "$REPO"
  echo "  ✓ ${name} set"
  echo
}

prompt_secret APPLE_TEAM_ID                  "10-char team ID (e.g. 77GHJHUGD4)"
prompt_secret APP_STORE_CONNECT_ISSUER_ID    "UUID from App Store Connect → Users and Access → Keys"
prompt_secret APP_STORE_CONNECT_KEY_ID       "10-char key ID from same page"
prompt_secret APP_STORE_CONNECT_PRIVATE_KEY  "base64 of the AuthKey_*.p8 file (\`base64 -i AuthKey_*.p8\`)"
prompt_secret IOS_DIST_CERT_P12              "base64 of the .p12 distribution cert"
prompt_secret IOS_DIST_P12_PASSWORD          "password for the .p12 above"

echo "────────────────────────────────────────────────────────────"
echo "Per-app secret (NEW — must be generated for io.dynolabs.cinova):"
echo
echo "  1. Apple Developer portal → Certificates, IDs & Profiles → Profiles"
echo "  2. Create a new App Store distribution profile for io.dynolabs.cinova"
echo "  3. Download the .mobileprovision file"
echo "  4. base64 -i Cinova_AppStore.mobileprovision  → paste below"
echo
prompt_secret IOS_DIST_PROVISION             "base64 of the .mobileprovision file"

echo "Done. Verify with: gh secret list -R ${REPO}"
