# Deploying somewhere other than the Pi

The Pi path is in [PI_DEPLOY_GUIDE](PI_DEPLOY_GUIDE.md). This guide covers a
personal VPS, the Google Cloud free tier and Fly.io. The app runs the same everywhere:

- **It is one binary.** The frontend is embedded, so nothing else needs to be served.
- **It keeps its data in `./db/sonic.db`**, relative to the working directory.
  That directory must sit on **persistent storage**, or every restart or
  redeploy wipes the household's data.
- **It listens on `:8080`** (plain HTTP), or on `:8443` when `TLS_CERT_FILE` and
  `TLS_KEY_FILE` are both set. The port is hardcoded: the `PORT` variable is ignored.

On first start the app logs the shared password once (see the README).
Read it from the service's logs, log in, and change it right away under
Settings (see the security section for why).

---

## ⚠️ Security: read this before exposing the app to the internet

The app was designed to be reachable **only over Tailscale** (see
[ADR-0007](../adr/0007-one-shared-password-over-tailscale.md)). Its security
model assumes nobody on the public internet can reach the login page:

- **One shared password protects everything.** There are no user accounts. Anyone
  who has it sees and edits all of the household's financial data.
- **Login attempts are not rate-limited.** A public login endpoint can be
  brute-forced. The only brake is bcrypt's cost, and the minimum password
  length is 8 characters.
- **The session cookie has no `Secure` flag.** If the site is ever reached over
  plain HTTP, the 30-day session cookie is sent in cleartext. Logging out does
  not revoke a stolen cookie. Changing the password does, because it
  invalidates every session.
- **The first-run password is written to the logs.** On a hosted service, the
  provider keeps those logs, and so can anyone with access to your dashboard.
  Change the password right after the first login.
- **The backup endpoint downloads the whole database** to any logged-in session.

Pick one of these options, from safest to least safe:

1. **Keep it private (recommended).** Run it on a VPS that joins your tailnet,
   and bind or firewall it so it is reachable only over Tailscale. This is the
   same model as the Pi in your home network.
2. **Put an authentication layer in front of it.** For example, Cloudflare
   Access / Zero Trust, or a reverse proxy with its own login. Internet traffic
   then has to authenticate before it reaches the app's login page.
3. **Expose it directly (not recommended).** If you do, at least:
   - serve it over **HTTPS only**, redirecting HTTP to HTTPS and turning on HSTS;
   - set a **long, random password** (20+ characters from a password manager),
     not a memorable one;
   - add rate limiting at the proxy if you can (e.g. `fail2ban` on 401s to
     `POST /api/login`, or your proxy's rate-limit module);
   - keep off-site backups. A compromised instance can also delete the data.

Please keep in mind that the app was not built or audited for the public internet.

---

## Building a binary for a server

Most VPSes and hosts run x86-64 Linux, not ARMv6 so after you built the frontend,
you have to cross-compile. CGO stays off because the SQLite driver is pure Go:

    pnpm --dir web install --frozen-lockfile
    pnpm --dir web build
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/api-amd64 ./cmd
    # ARM VPS (Hetzner CAX, Oracle Ampere, …): GOARCH=arm64

Fly.io builds from a Dockerfile: save this as `Dockerfile` at the
repo root.

```dockerfile
FROM node:24-slim AS web
WORKDIR /src
RUN corepack enable
COPY web/ web/
RUN pnpm --dir web install --frozen-lockfile && pnpm --dir web build

FROM golang:1.27 AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/cmd/static/ cmd/static/
RUN CGO_ENABLED=0 go build -o /sonic-finance ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /sonic-finance /app/sonic-finance
# /app/db must be a mounted persistent volume, see below
EXPOSE 8080
ENTRYPOINT ["/app/sonic-finance"]
```

> The distroless image runs as a non-root user (uid 65532). If the mounted
> volume isn't writable by that user, the app exits on start with a
> permission error. Either fix the volume's ownership, or switch the final
> stage to `debian:12-slim` and run as root.

---

## Personal VPS

Any small Linux VPS works, since the app needs very little (PI Zero W has 512 MB RAM and 1 Ghz single core cpu). The layout mirrors the Pi's (`user-data.example.yaml`).

1. **Create a user and a directory:**

       sudo useradd --system --create-home --shell /usr/sbin/nologin sonic
       sudo mkdir -p /opt/sonic-finance
       sudo chown sonic:sonic /opt/sonic-finance

2. **Copy the binary over:**

       scp build/api-amd64 <user>@<vps>:/tmp/sonic-finance
       ssh <user>@<vps> 'sudo install -o sonic -g sonic -m 755 /tmp/sonic-finance /opt/sonic-finance/sonic-finance'

3. **Add a systemd unit** at `/etc/systemd/system/sonic-finance.service`:

       [Unit]
       Description=Sonic Finance API
       After=network-online.target
       Wants=network-online.target

       [Service]
       User=sonic
       Group=sonic
       WorkingDirectory=/opt/sonic-finance
       ExecStart=/opt/sonic-finance/sonic-finance
       Restart=always
       RestartSec=5

       [Install]
       WantedBy=multi-user.target

   Then run:

       sudo systemctl daemon-reload
       sudo systemctl enable --now sonic-finance
       journalctl -u sonic-finance | grep "first run"   # the shared password

   The database is created at `/opt/sonic-finance/db/sonic.db`.

4. **Make it reachable.** Choose one:

   - **Tailscale only (recommended):** install Tailscale
     (`curl -fsSL https://tailscale.com/install.sh | sh && sudo tailscale up`),
     then firewall port 8080 off from the internet:

         sudo ufw default deny incoming
         sudo ufw allow OpenSSH
         sudo ufw allow in on tailscale0
         sudo ufw enable

     Open `http://<vps-tailscale-name>:8080`. For HTTPS and PWA install, the
     `mkcert` steps in [PI_DEPLOY_GUIDE](PI_DEPLOY_GUIDE.md#tls-certificate)
     work unchanged.
     Check your provider's own cloud firewall too.

   - **Public, behind Caddy:** read the security section first! Keep the app
     firewalled to localhost, and let Caddy fetch a Let's Encrypt certificate
     and terminate TLS. `/etc/caddy/Caddyfile`:

         finance.example.com {
             reverse_proxy 127.0.0.1:8080
             header Strict-Transport-Security "max-age=31536000"
         }

     Open only 80/443 in `ufw` (`sudo ufw allow 80,443/tcp`), not 8080.
     Caddy redirects HTTP to HTTPS on its own.

5. **Updating:** copy the new binary over the old one and run
   `sudo systemctl restart sonic-finance`. Schema migrations run on start.

**Backups:** use Settings → backup, or copy the database on the server with
`sqlite3 /opt/sonic-finance/db/sonic.db ".backup /somewhere/sonic.db"`.
Don't `cp` the live file: WAL mode keeps recent writes in `sonic.db-wal`.

---

## Google Cloud (GCP) free tier

The [free tier](https://docs.cloud.google.com/free/docs/free-cloud-features#compute)
gives you, every month:

- one **`e2-micro`** VM (2 shared vCPUs, 1 GB RAM), only in `us-west1`,
  `us-central1` or `us-east1`;
- **30 GB of standard persistent disk**;
- **1 GB of outbound traffic** from North America.

That is a small VPS, so the [Personal VPS](#personal-vps) steps apply with a few
GCP-specific changes. It is "free" only if you stay inside these limits:

- **The boot disk must be *Standard persistent disk*.** The console's default
  is *Balanced*, which isn't covered by the free tier and gets billed.
- **An external IPv4 address isn't free.** It costs $0.005/hour, about $3.60 a
  month, and the free tier covers only one hour of it per month
  ([network pricing](https://cloud.google.com/vpc/network-pricing#ipaddress)).
  Tailscale needs outbound internet access, so in practice you either pay that
  or go IPv6-only (see below).
- **Set a budget alert** (Billing → Budgets & alerts, e.g. $1) so a
  misconfiguration can't run up a bill without you noticing.

1. **Create the VM** (or do the same in the console under Compute Engine → VM
   instances). The zone must be in one of the three free regions:

       gcloud compute instances create sonic-finance \
         --zone=us-central1-a \
         --machine-type=e2-micro \
         --image-family=debian-12 --image-project=debian-cloud \
         --boot-disk-size=30GB --boot-disk-type=pd-standard

   The boot disk *is* the persistent storage. `db/` lives on it and survives
   reboots and stops.

2. **Copy the binary.** An e2-micro is x86-64, so use `build/api-amd64`:

       gcloud compute scp build/api-amd64 sonic-finance:/tmp/sonic-finance --zone=us-central1-a
       gcloud compute ssh sonic-finance --zone=us-central1-a

3. **On the VM**, follow [Personal VPS](#personal-vps) steps 1–3: the user, the
   install, and the systemd unit.

4. **Make it reachable over Tailscale.** Install it as in the Personal VPS
   step 4 and open `http://<vm-tailscale-name>:8080`.
   You don't need `ufw`: GCP's default VPC firewall already blocks every
   inbound port except SSH (22), ICMP and internal traffic, so 8080 is closed
   to the internet unless you add a rule for it. **Don't add one.**
   Once Tailscale works, you can also remove the `default-allow-ssh` rule and
   SSH over your tailnet instead (`ssh <user>@<vm-tailscale-name>`, or
   Tailscale SSH with `sudo tailscale up --ssh`).

   If you really want it public, use the Caddy option from the Personal VPS
   section and add a firewall rule for 80/443 only. **Read the security section
   first.**

**Avoiding the IPv4 charge (advanced, untested with this app):** GCP doesn't
charge for external IPv6 addresses. A VM on a dual-stack subnet with an
external IPv6 address and *no* external IPv4 address costs nothing extra.
Tailscale can run over IPv6, but anything reachable only over IPv4 (for
example GitHub downloads) won't work from that VM, so copy the binary over
with `gcloud compute scp` and check each step yourself.

**Traffic:** 1 GB a month is plenty for a household using the app, but the
first load of the frontend on each new device counts against it, and so does
downloading backups.

---

## Fly.io

1. Add the Dockerfile above, install `flyctl`, and run `fly launch --no-deploy`
   from the repo root. Let it create the app, but no databases.
2. Create a volume in the same region as the app:

       fly volumes create sonic_data --size 1

3. Edit `fly.toml` so it has at least these sections:

       [mounts]
         source = "sonic_data"
         destination = "/app/db"

       [http_service]
         internal_port = 8080
         force_https = true
         auto_stop_machines = "stop"
         auto_start_machines = true
         min_machines_running = 0

4. Deploy, and keep it to **one machine**, because SQLite can't be shared across machines:

       fly deploy
       fly scale count 1
       fly logs | grep "first run"    # the shared password

Your app is now public at `https://<app>.fly.dev`, and **the security section
applies**. For a private setup, remove the `[http_service]` section's public
IPs (`fly ips list` / `fly ips release`) and reach it over Fly's WireGuard
(`fly wireguard create`), or run Tailscale inside the container. Both keep
the Tailscale-style model from ADR-0007.

Snapshots: Fly takes daily volume snapshots, but still download a backup
from Settings now and then.

---

## Checklist, wherever you deploy

- [ ] `db/` sits on persistent storage (restart the service and check the data is still there)
- [ ] Shared password changed after the first login
- [ ] Not publicly reachable, **or** HTTPS-only, a long random password, and ideally an auth layer in front
- [ ] Only one instance running
- [ ] Off-site backups are being downloaded


> [!warning]
> this guide is not kept up to date and may contain errors or inconsistencies. 
> Do your own research (or just get a Pi)
