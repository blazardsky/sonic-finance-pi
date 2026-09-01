# One shared password, plain HTTP, reached over Tailscale

The app serves plain HTTP and is reached through Tailscale, which provides the transport encryption. Access is guarded by a single shared password (bcrypt-hashed in the settings table) and a 30-day signed session cookie. There are no usernames and no per-person accounts.

Per-person accounts contradict ADR-0001, where Payer is a label rather than an identity. No auth at all was the original plan and was rejected once Tailscale entered the picture: the password protects against anyone else on the tailnet and against an unlocked phone, not against network sniffing.

## Consequences

**This decision is invalid the moment the app is reachable from the public internet.** At that point it needs TLS (`tailscale cert`, or a reverse proxy) and login rate-limiting before anything else.
