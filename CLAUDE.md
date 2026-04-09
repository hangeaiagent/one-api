# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

One API is a unified LLM API gateway written in Go with a React frontend. It exposes an OpenAI-compatible API (`/v1/*`) that proxies requests to 25+ LLM providers (OpenAI, Anthropic, Gemini, AWS Bedrock, Azure, Ali, Baidu, DeepSeek, etc.). It also serves a management dashboard (`/api/*`) for users, channels, tokens, and quotas.

## Build & Development Commands

```bash
# Build Go binary
go build -ldflags "-s -w" -o one-api

# Run tests with coverage
go test -cover -coverprofile=coverage.txt ./...

# Run a single test
go test ./relay/adaptor/aws/llama3/ -run TestName

# Build frontend (from web/ directory)
cd web/default && npm install && REACT_APP_VERSION=local npx react-scripts build

# Docker build (multi-stage: frontend + backend)
docker build -t one-api .

# Docker Compose (full stack with MySQL + Redis)
docker-compose up
```

The Go module requires Go 1.20+. CI uses Go 1.22. Commit messages are linted with commitlint.

## Architecture

### Request Flow
1. Client sends OpenAI-format request to `/v1/chat/completions` (or other endpoints)
2. `middleware/auth.go` authenticates via Bearer token
3. `middleware/rate-limit.go` enforces rate limits
4. `middleware/distributor.go` selects a channel (load balancing across providers in a group)
5. `controller/relay.go` dispatches to the appropriate provider adaptor with retry logic (retries on 429/5xx)
6. Provider adaptor converts request/response between OpenAI format and provider-specific format
7. `relay/billing/` counts tokens and deducts quota
8. `model/log.go` records the request

### Key Packages

- **`relay/adaptor/`** — Provider adaptors implementing `Adaptor` interface (`interface.go`). Each subdirectory (e.g., `openai/`, `anthropic/`, `gemini/`) handles request/response conversion for one provider.
- **`relay/controller/`** — Relay logic for text completions, image generation, audio, embeddings.
- **`relay/channeltype/`** — Channel type constants and URL definitions for all providers.
- **`relay/model/`** — Shared data structures (OpenAI request/response formats).
- **`controller/`** — Management API handlers (users, channels, tokens, billing, OAuth).
- **`model/`** — GORM database models and cache layer. Supports SQLite (default), MySQL, PostgreSQL.
- **`middleware/`** — Auth, rate limiting, channel distribution, CORS, logging.
- **`common/config/`** — All configuration via environment variables.
- **`monitor/`** — Channel health tracking, auto-disable on failures, metrics.
- **`router/`** — Route registration. `api.go` for management routes, `relay.go` for OpenAI-compatible routes.

### Adaptor Interface

To add a new LLM provider, implement `relay/adaptor/Adaptor`:
```go
type Adaptor interface {
    Init(meta *meta.Meta)
    GetRequestURL(meta *meta.Meta) (string, error)
    SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error
    ConvertRequest(c *gin.Context, relayMode int, request *model.GeneralOpenAIRequest) (any, error)
    ConvertImageRequest(request *model.ImageRequest) (any, error)
    DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error)
    DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode)
    GetModelList() []string
    GetChannelName() string
}
```
Then register the adaptor in `relay/adaptor/` and add the channel type constant in `relay/channeltype/`.

### Frontend

Three React themes in `web/` (`default/`, `berry/`, `air/`). Built assets are embedded into the Go binary via `//go:embed`. Theme is selected via `THEME` env var.

### Database

GORM auto-migrates models on startup (`model/main.go`). The `Ability` table maps which models are available on which channels, enabling the distributor to select appropriate channels per request.

## Key Environment Variables

- `PORT` — Server port (default: 3000)
- `SQL_DSN` — MySQL/PostgreSQL connection string (omit for SQLite)
- `REDIS_CONN_STRING` — Redis connection (omit for in-memory cache)
- `SESSION_SECRET` — Required for production session security
- `RELAY_TIMEOUT` — Upstream request timeout
- `GLOBAL_API_RATE_LIMIT` — API rate limit (default: 480 per 3 min)
- `MEMORY_CACHE_ENABLED` — Enable in-memory caching
- `NODE_TYPE=slave` — Run as slave node in multi-instance deployment
