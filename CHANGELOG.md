# Changelog

All notable changes to this project are documented here.

## 2026-08-27
- Removed a hardcoded Redis credential from `main.go` (now read from `REDIS_URL`), ran gofmt cleanup, bumped `go-redis` to v9.22.0, removed a stray committed junk file, and added unit tests for the previously-untested `RateLimiter`/`MultiLimiter` core.
