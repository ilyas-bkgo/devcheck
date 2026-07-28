# devcheck examples

Copy the template closest to your project into its root, then customize it:

```bash
cp examples/web-dev-node.yaml devcheck.yaml
devcheck
```

Available templates:

- `web-dev-node.yaml`: Node.js, pnpm, Docker, and PostgreSQL on port 5432.
- `python-backend.yaml`: Python, Poetry, Redis, and an HTTP health endpoint on port 8000.
- `go-microservices.yaml`: Go 1.25, `protoc`, `golangci-lint`, Docker, and an HTTP health endpoint on port 8080.

The HTTP URLs are examples. Change them to match the service your project exposes.
