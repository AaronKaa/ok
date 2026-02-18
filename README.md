# ok
[![CI/CD](https://github.com/AaronKaa/ok/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/AaronKaa/ok/actions/workflows/docker-publish.yml)

A lightweight health check aggregator that runs inside your container. Point it at your dependencies, and it'll give you a single endpoint that reports whether everything is healthy.

## Why?

I wanted all of my health checks in one place and in the same format... I found it nice to have a mini health dashboard for projects during dev too... 

## Quick Start

Set up your checks via environment variables and run:

```bash
export CHECKS='[
  {"title": "API", "url": "http://api:8080/health"},
  {"title": "Database", "url": "http://db:5432/health", "interval": "60s"}
]'

./ok
```

Then hit `http://localhost:8080/health` to see the aggregate status.

## Docker Compose

```yaml
services:
  ok:
    image: aarcarr/ok:latest
    ports:
      - "8080:8080"
    environment:
      - CHECKS_API={"title":"API","url":"http://api:8080/health"}
      - CHECKS_DB={"title":"Database","url":"http://db:5432/health","interval":"60s"}
      - CHECKS_REDIS={"title":"Redis","url":"http://redis:6379/health","critical":false}
```

Or with an env file:

```yaml
services:
  ok:
    image: aarcarr/ok:latest
    ports:
      - "8080:8080"
    env_file:
      - .env
```

With Docker native health checks:

```yaml
services:
  mysql:
    image: mysql:8
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 3

  ok:
    image: aarcarr/ok:latest
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - CHECKS_MYSQL={"title":"MySQL","type":"docker","container":"mysql"}
      - CHECKS_API={"title":"API","url":"http://api:8080/health"}
```

## Configuration

### Server Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `REFRESH_INTERVAL` | `10s` | HTML auto-refresh interval (set to `0` to disable) |

### Defining Checks

Checks are configured through environment variables. You can use `CHECKS` for an array of checks, or `CHECKS_<name>` for individual checks:

```bash
# Array of checks
CHECKS='[{"title": "API", "url": "http://api/health"}]'

# Individual checks (useful for cleaner compose files)
CHECKS_DB='{"title": "Database", "url": "http://db:5432/health"}'
CHECKS_REDIS='{"title": "Cache", "url": "http://redis:6379/health"}'
```

### Check Types

ok supports two types of health checks:

| Type | Description |
|------|-------------|
| `http` | HTTP endpoint checks (default) |
| `docker` | Docker container native health checks |

### HTTP Check Options

| Field | Default | Description |
|-------|---------|-------------|
| `title` | *required* | Display name for the check |
| `type` | `http` | Check type |
| `url` | *required* | HTTP(S) endpoint to probe |
| `method` | `GET` | HTTP method |
| `interval` | `30s` | How often to run the check |
| `expect_status` | `200` | Expected HTTP status code |
| `json_path` | | JSON path to extract (uses [gjson](https://github.com/tidwall/gjson) syntax) |
| `json_equals` | | Expected value at the JSON path |
| `critical` | `true` | Whether failure should mark the aggregate as failed |
| `retries` | `0` | Number of failures before marking as failed |

### Docker Check Options

| Field | Default | Description |
|-------|---------|-------------|
| `title` | *required* | Display name for the check |
| `type` | | Must be `docker` |
| `container` | *required* | Container name or ID |
| `interval` | `30s` | How often to run the check |
| `critical` | `true` | Whether failure should mark the aggregate as failed |
| `retries` | `0` | Number of failures before marking as failed |

### HTTP Check Examples

Basic health check:
```json
{"title": "API", "url": "http://api:8080/health"}
```

Check with JSON validation:
```json
{
  "title": "Database",
  "url": "http://db:5432/status",
  "json_path": "status",
  "json_equals": "healthy"
}
```

Non-critical check (won't fail the aggregate):
```json
{
  "title": "Analytics",
  "url": "http://analytics/ping",
  "critical": false
}
```

Check with retries (tolerates brief outages):
```json
{
  "title": "External API",
  "url": "https://api.example.com/health",
  "interval": "1m",
  "retries": 2
}
```

### Docker Check Examples

Monitor a container's native Docker health check:
```json
{
  "title": "MySQL",
  "type": "docker",
  "container": "mysql"
}
```

Docker checks read the container's health status from Docker's API. The container must have a `HEALTHCHECK` defined in its Dockerfile or compose file. If no healthcheck is configured, ok considers a running container as healthy.

## Endpoints

### GET /health

Returns the aggregate health status. Supports both HTML and JSON:

- **HTML** (default): Human-readable status page with auto-refresh
- **JSON**: Machine-readable status for monitoring systems

To get JSON, either:
- Set `Accept: application/json` header
- Add `?format=json` query parameter

### Response Codes

| Status | HTTP Code | Meaning |
|--------|-----------|---------|
| `pass` | 200 | All checks passing |
| `degraded` | 200 | Non-critical checks failing, or within retry threshold |
| `fail` | 503 | Critical check failed |

### JSON Response

```json
{
  "status": "pass",
  "timestamp": "2025-01-15T10:30:00Z",
  "checks": [
    {
      "id": "db",
      "title": "Database",
      "status": "pass",
      "critical": true,
      "last_check": "2025-01-15T10:29:55Z",
      "age_seconds": 5.2,
      "duration_ms": 12.5,
      "message": "status 200",
      "consecutive_failures": 0
    }
  ]
}
```

## Status Logic

A check can be in one of three states:

- **pass**: Check succeeded
- **degraded**: Check failed but is non-critical, or still within retry threshold
- **fail**: Critical check failed beyond retry count

The aggregate status follows the worst state: if any critical check is failing, the aggregate is `fail`. If any check is degraded (but none failing), the aggregate is `degraded`. Otherwise, it's `pass`.

## Webhooks

Send notifications when health status changes. Webhooks fire when the aggregate status transitions (e.g., pass → fail, fail → pass).

### Webhook Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBHOOK_URL` | | URL to POST notifications to |
| `WEBHOOK_CONTINUOUS` | `false` | If `true`, fire on every check while unhealthy (not just on state change) |
| `WEBHOOK_AUTH_TYPE` | `none` | Authentication type: `none`, `basic`, `bearer`, or `header` |
| `WEBHOOK_AUTH_USER` | | Username for basic auth |
| `WEBHOOK_AUTH_PASS` | | Password for basic auth |
| `WEBHOOK_AUTH_TOKEN` | | Token for bearer auth |
| `WEBHOOK_AUTH_HEADER_NAME` | | Header name for custom header auth |
| `WEBHOOK_AUTH_HEADER_VALUE` | | Header value for custom header auth |

### Webhook Payload

```json
{
  "status": "fail",
  "previous_status": "pass",
  "timestamp": "2025-01-15T10:30:00Z",
  "failed_checks": [
    {
      "id": "db",
      "title": "Database",
      "status": "fail",
      "critical": true,
      "message": "connection refused",
      "consecutive_failures": 3,
      "duration_ms": 150.5
    }
  ]
}
```

### Authentication Examples

No authentication:
```bash
WEBHOOK_URL=https://hooks.example.com/health
```

Basic auth:
```bash
WEBHOOK_URL=https://hooks.example.com/health
WEBHOOK_AUTH_TYPE=basic
WEBHOOK_AUTH_USER=myuser
WEBHOOK_AUTH_PASS=mypassword
```

Bearer token:
```bash
WEBHOOK_URL=https://hooks.example.com/health
WEBHOOK_AUTH_TYPE=bearer
WEBHOOK_AUTH_TOKEN=my-secret-token
```

Custom header (e.g., API key):
```bash
WEBHOOK_URL=https://hooks.example.com/health
WEBHOOK_AUTH_TYPE=header
WEBHOOK_AUTH_HEADER_NAME=X-API-Key
WEBHOOK_AUTH_HEADER_VALUE=my-api-key
```

### Docker Compose with Webhook

```yaml
services:
  ok:
    image: aarcarr/ok:latest
    ports:
      - "8080:8080"
    environment:
      - CHECKS_API={"title":"API","url":"http://api:8080/health"}
      - WEBHOOK_URL=https://hooks.slack.com/services/xxx/yyy/zzz
      - WEBHOOK_AUTH_TYPE=none
```
