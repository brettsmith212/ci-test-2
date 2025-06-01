# Agent Memory

## Project Information
- **Language**: Go
- **Framework**: Gin (HTTP server), GORM (database), Cobra (CLI)
- **Database**: SQLite
- **API**: RESTful API with `/api/v1/` prefix

## Commands
- **Build**: `go build ./...`
- **Test**: `go test ./...`
- **Start Server**: `go run cmd/orchestrator/main.go`
- **Start CLI**: `go run cmd/ampx/main.go`
- **Start Worker**: `go run cmd/worker/main.go`

## API Testing Guidelines

### Curl POST Best Practices
**Always use these patterns to avoid shell hanging issues:**

**Recommended (pipe from echo):**
```bash
echo '{"repo": "https://github.com/test/repo.git", "prompt": "Fix the bug"}' | curl -s -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d @-
```

**Alternative (single line with escaped quotes):**
```bash
curl -s -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d "{\"repo\": \"https://github.com/test/repo.git\", \"prompt\": \"Fix the bug\"}"
```

**Key points:**
- Always use correct API paths (`/api/v1/tasks` not `/tasks`)
- Use `-s` flag for silent mode
- Avoid multiline heredoc syntax which causes commands to hang
- Check server logs if requests seem to hang

### API Endpoints
- **Health**: `GET /health`
- **Ping**: `GET /api/v1/ping`
- **Create Task**: `POST /api/v1/tasks`
- **List Tasks**: `GET /api/v1/tasks`
- **Get Task**: `GET /api/v1/tasks/{id}`
- **Update Task**: `PATCH /api/v1/tasks/{id}`
- **Active Tasks**: `GET /api/v1/tasks/active`

## Code Style
- Follow existing Go conventions
- Use GORM for database operations
- Use Gin for HTTP handlers
- Use structured logging
- Follow the service layer pattern
