#!/bin/bash
# Lives on the Pi at /opt/sonic-finance/ci-deploy.sh (copy it there by hand —
# it can't deploy itself). Runs only via the CI deploy SSH key, which
# authorized_keys forces to this one command regardless of what the client
# asks for — see PI_DEPLOY_GUIDE.md.
#
# Reads the new binary off stdin, sanity-checks it's really an ELF binary
# before touching anything, then swaps it in with a rename (safe even while
# the old one is running: ETXTBSY only hits an open()/write(), never a
# rename()) and restarts the service.
set -euo pipefail
BIN=/opt/sonic-finance/sonic-finance
TMP="$BIN.incoming"

cat > "$TMP"

magic=$(head -c4 "$TMP" | od -An -tx1 | tr -d ' \n')
if [ "$magic" != "7f454c46" ]; then
  echo "refusing: not an ELF binary (got magic $magic)" >&2
  rm -f "$TMP"
  exit 1
fi

chmod +x "$TMP"
mv "$TMP" "$BIN"
sudo systemctl restart sonic-finance
echo "deployed and restarted"
