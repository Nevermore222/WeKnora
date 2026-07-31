# Xelora Pi Workspace Sidecar

Personal-only NDJSON stdio runtime for cwd-scoped Skill / file / shell tools.

## Protocol

- One JSON object per line on stdin/stdout
- Handshake: `{"type":"handshake","protocol_version":1}` → `{"type":"handshake_ok","protocol_version":1,"ok":true}`
- Tool call: see `internal/piruntime/protocol.go`

## Develop

```bash
npm install
npm run build
npm start
```

Node ≥ 22 required.

## Note

v1 implements confined Node `fs` / `child_process` behind the Pi workspace RPC surface.
Pi coding-agent SDK tool factories can replace internals later without changing Go Facade contracts.
