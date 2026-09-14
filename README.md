# Sonic Finance | GO Backend API + Vite Frontend

This is the code for the Sonic Finance App, a simple expenses and icomes tracker for households (multiple people, shared space). 

## Stack

- Go 1.27
- modernc.org/sqlite
- vite
- react

## Target

Raspberry PI Zero W 1st gen | ARMv6

---

To prevent SD card corruption and prolong lifespan: Enable Write-Ahead Logging (PRAGMA journal_mode = WAL;) and PRAGMA synchronous = NORMAL; in SQLite.

## First run

The app generates a shared password the first time it starts and logs it once:

    first run: the shared password is HCQJS3BE… — write it down

Write it down: it is logged once and never shown again. If it is lost,
`DELETE FROM setting WHERE key = 'password_hash'` and restart to get a new one.

## Development

`./build.sh` builds the frontend into `cmd/static/`, which the binary embeds, then
compiles both binaries into the gitignored `build/`: `build/api-armv6` is the
single file to copy to the Pi, and `build/sonic-finance-pi` is the same code
for this machine. The dev loop — the Go server and the Vite dev server side by
side — is in `web/README.md`.


### Localhost testing

Run the command `pnpm --dir web dev` for the Vite app, run the command `go run ./cmd` for the Go server.

## Deploying to the Pi

There is no CD — a new binary reaches the Pi by hand. Two ways to get one:

**A) Build it yourself:** `./build.sh` produces `build/api-armv6`, ready to copy over.

**B) Grab a CI build:** pushing a tag alone builds nothing — `.github/workflows/release-pi-zero.yml`
only runs `on: release: published`, so a GitHub Release has to actually be
created from that tag:

    git tag vX.Y                 # X.Y — whatever this release is for you
    git push origin vX.Y
    gh release create vX.Y       # publishes it, which is what triggers the build

Wait for the workflow to finish (`gh run watch`, or check the Actions tab),
then the binary is attached to the release as `sonic-finance-armv6`:

    curl -L -o sonic-finance-armv6 \
      https://github.com/blazardsky/sonic-finance-pi/releases/download/vX.Y/sonic-finance-armv6
      # vX.Y — the tag you just released

Either way, copying it over and restarting is the same, and is the one part
of this that is genuinely yours to fill in — this repo carries no systemd
unit or deploy script, so how the binary is run on your Pi is whatever you
set up when you first installed it:

    scp sonic-finance-armv6 <user>@<pi-host>:<path-to-the-running-binary>
      # <user>@<pi-host> — however you already SSH in: an IP, `raspberrypi.local`,
      # a Tailscale name, whatever's in your own ~/.ssh/config.
      # <path-to-the-running-binary> — wherever the binary already lives on the
      # Pi. Match the existing name/path exactly if something (a systemd unit,
      # a login script, a crontab @reboot line) hardcodes it — otherwise the
      # thing that starts it on boot will look in the wrong place.

    ssh <user>@<pi-host> 'systemctl restart <service-name>'
      # only if it runs as a systemd service — <service-name> is whatever you
      # named it (`sudo systemctl status` on the Pi will show it if you forget).
      # Running it by hand instead (a screen/tmux session, a plain foreground
      # process)? Kill that process and start the new binary the same way you
      # started the old one.

## How to install a certificate (TLS)

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