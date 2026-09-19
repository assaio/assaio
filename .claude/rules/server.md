---
paths:
  - "internal/server/**"
  - "internal/cli/serve.go"
  - "internal/cli/sync*.go"
  - "internal/share/**"
---

# Team server, sync and shared artifacts

- **The team server is an MVP, and every surface says so**: no TLS, loopback by default, one
  shared bearer token or per-member secrets (`server.members`), `/healthz` the only open route.
  RBAC, rotation, resumable sync, retention and a restore drill are roadmap milestone 4 gates.
- **Nothing content-bearing crosses the wire**: pseudonymous identity, token counts, model names,
  timestamps, content-free counts. Labels never leave the machine (ADR 0006). A new field that
  leaves the machine changes `PRIVACY.md` and `docs/threat-model.md` in the same commit.
- **`sync` warns on cleartext to a non-loopback host** and never uses `http.DefaultClient`; a 401
  is reported as auth failure, never retried with another identity.
- **Member session bars scale by sessions only**, never lines or cost — a leaderboard by another
  name is still a refusal.
- **A shareable artifact (`share`) is structurally redacted** (ADR 0014): no figure originates in
  the renderer, every frame carries its own limits because the image outlives its caption.
