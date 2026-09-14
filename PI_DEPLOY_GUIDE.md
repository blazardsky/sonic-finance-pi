# Deploying to the Pi

## Automatic (the normal path)

Publishing a GitHub Release deploys itself. `.github/workflows/release-pi-zero.yml`
builds the ARMv6 binary, attaches it to the release, then joins the Pi's
Tailscale network and pushes + restarts it over a restricted SSH key. All you
do:

    git tag vX.Y && git push origin vX.Y
    gh release create vX.Y

Tagging alone builds nothing — the workflow only runs `on: release: published`.
Watch it happen with `gh run watch`, or check the Actions tab.

### One-time setup behind this (already done for this repo)

- **`scripts/setup-tailscale-ci.sh`** — an interactive wizard that creates the
  Tailscale OAuth client the workflow uses to join the tailnet, and stores
  its credentials as the `TS_OAUTH_CLIENT_ID` / `TS_OAUTH_CLIENT_SECRET`
  GitHub secrets. Re-run it if those ever need rotating.
- **`scripts/ci-deploy.sh`** — lives on the Pi at `/opt/sonic-finance/ci-deploy.sh`
  (copy it there by hand; it can't deploy itself). A dedicated SSH key on the
  Pi is restricted, via a forced command in `~/.ssh/authorized_keys`, to
  running only this script — no shell, no other command — so the
  `PI_DEPLOY_SSH_KEY` GitHub secret is harmless even leaked. The script only ever accepts a real ELF binary before swapping it in and restarting the service.

## Manual (fallback, or a different install)

Get a binary — either build it locally, or grab a published release's asset:

    ./build.sh                                    # -> build/api-armv6
    gh release download vX.Y --pattern sonic-finance-armv6

Copy it to wherever it runs and restart:

    scp <binary> <user>@<host>:<path>              # <path> = wherever the binary already lives
    ssh <user>@<host> 'systemctl restart <service>' # or however yours is supervised

There's no install script for a fresh Pi — `<path>` and `<service>` depend on
how that install was first set up.

## TLS certificate

By default the server runs plain HTTP on `:8080` — fine for use over Tailscale,
which already encrypts the connection. But installing the frontend as a PWA
("Add to Home Screen") requires a *secure context*, i.e. HTTPS, before Android
Chrome will offer the install prompt (iOS Safari doesn't require this and works
without any of the below).

We don't use `tailscale cert` for this: it issues certificates through Let's
Encrypt, a publicly trusted CA, which means the certificate — and therefore
your device's Tailscale hostname — gets published to public Certificate
Transparency logs forever. Instead, use `mkcert` to run your own private CA
that nothing outside your devices ever needs to trust:

1. **On your own machine** (not the Pi/VPS), install and set up `mkcert`:

       brew install mkcert   # or see https://github.com/FiloSottile/mkcert
       mkcert -install

   This creates a local root CA and trusts it on your machine. Its private key
   never leaves this machine.

2. **Generate a certificate** for the server's actual Tailscale name/IP:

       mkcert raspberrypi.your-tailnet.ts.net 100.x.y.z

   This produces two files, e.g. `raspberrypi.your-tailnet.ts.net+1.pem` (cert)
   and `raspberrypi.your-tailnet.ts.net+1-key.pem` (key).

3. **Copy the cert and key to the Pi/VPS** (e.g. via `scp`), then start the
   server with both env vars set:

       TLS_CERT_FILE=/path/to/cert.pem TLS_KEY_FILE=/path/to/key.pem ./api-armv6

   The server now listens on `:8443` with TLS. Leaving either variable unset
   keeps the previous plain-HTTP behavior on `:8080`.

4. **Trust the root CA on every device** that will use the app (phone,
   laptop) — not the leaf certificate, just the CA. Find it with
   `mkcert -CAROOT` on the machine from step 1, then copy `rootCA.pem` to each
   device and install it as a trusted certificate (iOS: AirDrop/email the file,
   install the profile, then also enable full trust for it under Settings →
   General → About → Certificate Trust Settings; Android: Settings → Security
   → Encryption & credentials → Install a certificate → CA certificate).

   This is a one-time step per device. Nothing is published anywhere, since
   `mkcert`'s CA is never chained to a publicly trusted root.
