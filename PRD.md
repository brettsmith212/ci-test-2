# PRD — **CI-Driven Background Agent Orchestrator (Amp Companion)**

---

## 1 Overview

Build a lightweight **Orchestrator API (Go)** plus a **CLI UI** that lets developers launch, monitor, pause, and resume CI-driven Amp tasks from a single terminal.
The orchestrator tracks branch names, thread-IDs, and CI runs, so users never lose context or restart scripts manually.

---

## 2 Goals & Non-Goals

|                                           | **Goal**                        | **Non-Goal** |
| ----------------------------------------- | ------------------------------- | ------------ |
| Self-service launch (`ampx start`)        | ✅                              |              |
| Persist task metadata (SQLite)            | ✅                              |              |
| Resume / continue tasks (`ampx continue`) | ✅                              |              |
| Poll GitHub Actions & fetch logs          | ✅                              |              |
| Support GitHub App auth                   | ✅                              |              |
| Web UI                                    | ❌ phase 2                      |
| Multi-CI vendor support                   | ❌ (GitHub Actions only)        |
| Parallel worker autoscaling               | ❌ (single worker goroutine OK) |

---

## 3 Personas & User Stories

| Persona       | Story                                                               | Acceptance Criteria                                                                   |
| ------------- | ------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| **Solo Dev**  | _“Kick off an Amp refactor before leaving office and check later.”_ | `ampx start` returns a Task-ID; `ampx ls` shows status; CI passes; PR opened.         |
| **Team Lead** | _“See all team tasks in one list and nudge a failing one.”_         | `ampx ls --all` lists everyone; `ampx continue <id> -m "try X"` queues a new attempt. |
| **SRE**       | _“Abort an infinite-looping task.”_                                 | `ampx abort <id>` sets status `aborted`; worker terminates.                           |

---

## 4 System Components

### 4.1 Orchestrator API (Go 1.22, Gin)

- `/tasks` POST — create
- `/tasks/{id}` GET/PATCH
- `/tasks` GET — list
- Persists to **SQLite** via `gorm.io`.

### 4.2 Worker

- Docker image: `golang:1.22-alpine + git + amp + gh + jq`.
- Reads one row, runs CI loop, updates status.

### 4.3 CLI (`ampx`)

- Cobra-based; sub-commands:
  - `start`, `list`, `logs`, `continue`, `abort`, `merge`

---

## 5 Data Model (`tasks` table)

| Field                       | Type                                                              | Notes |
| --------------------------- | ----------------------------------------------------------------- | ----- |
| `id`                        | TEXT PK (ULID)                                                    |
| `repo`                      | TEXT                                                              |
| `branch`                    | TEXT                                                              |
| `thread_id`                 | TEXT                                                              |
| `prompt`                    | TEXT — latest prompt to Amp                                       |
| `status`                    | ENUM `queued/running/retrying/needs_review/success/aborted/error` |
| `ci_run_id`                 | INTEGER (GitHub run)                                              |
| `attempts`                  | INT                                                               |
| `summary`                   | TEXT                                                              |
| `created_at` / `updated_at` | TIMESTAMP                                                         |

---

## 6 API Spec

| Method  | Path          | Body / Params                                       | Returns            |
| ------- | ------------- | --------------------------------------------------- | ------------------ |
| `POST`  | `/tasks`      | `{repo, prompt}`                                    | `201 {id, branch}` |
| `GET`   | `/tasks`      | `?status=`                                          | list rows          |
| `GET`   | `/tasks/{id}` |                                                     | full row           |
| `PATCH` | `/tasks/{id}` | `{action:"continue", prompt}` or `{action:"abort"}` | 204                |

---

## 7 Key Workflows

### 7.1 Start

1. CLI `ampx start --repo X --task "…"`.
2. Orchestrator:
   - `thread_id = amp threads new`
   - `branch = amp/<slug>`
   - Insert row `status=queued`.
3. Dispatcher goroutine spawns Worker with row ID env-var.

### 7.2 Worker Loop (pseudocode)

````go
for attempts < MaxRetries {
    diff := callAmp(threadID, prompt)
    applyPatch(diff); push(branch)

    runID, ok := waitForCI(branch, sha)
    if !ok {
        update(status="error"); return
    }
    if ciGreen(runID) {
        openPR(); update(status="success"); return
    }
    logs := fetchFailLogs(runID)
    prompt = fmt.Sprintf("CI failed:\n```%s```\nFix and retry.", slice(logs,0,4000))
    attempts++
    update(status="retrying", prompt=prompt, attempts=attempts)
}
update(status="needs_review", summary="max retries hit")
````

---

## 8 CLI UX Examples

```bash
# Kick off a task
ampx start --repo git@github.com:acme/api.git \
           --task "Migrate Mocha tests to Vitest"

# Follow live
ampx ls -w                 # watch mode

# View failing logs
ampx logs 01HYZ3...        # streams worker + CI excerpt

# Continue with new prompt
ampx continue 01HYZ3... -m "Restore skipped test and fix bug"

# Abort
ampx abort 01HYZ3...
```

---

## 9 Security & Auth

- Use a **GitHub App** with scopes:
  `contents:write`, `pull_requests:write`, `actions:read`.
- App’s private key mounted into worker container.
- Orchestrator only exposes localhost by default; use reverse-proxy if multi-user.

---

## 10 MVP Success Metrics

| Metric                                       | Target           |
| -------------------------------------------- | ---------------- |
| Tasks launched & finished without manual git | ≥ 5 in demo repo |
| Mean “green-CI time” vs manual workflow      | ≤ +20 %          |
| Lines of code in orchestrator ≤              | 600 Go           |

---

## 11 Milestones

| Week | Deliverable                                                                      |
| ---- | -------------------------------------------------------------------------------- |
| 1    | Repo scaffolding, SQLite schema, `POST /tasks`, `ampx start`, single-shot worker |
| 2    | CI polling, retry loop, `ls` + `logs`                                            |
| 3    | `continue` & `abort`, auth via GitHub App                                        |
| 4    | README demo, internal dog-food on real repo                                      |

---

## 12 Reference Code Snippets

### 12.1 Create Task (Go Gin)

```go
type NewTask struct {
    Repo  string `json:"repo"  binding:"required"`
    Prompt string `json:"prompt" binding:"required"`
}

func createTask(c *gin.Context) {
    var body NewTask
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(400, gin.H{"error": err.Error()}); return
    }
    id := ulid.Make().String()
    branch := fmt.Sprintf("amp/%s", id[:6])
    thread := newAmpThread()                   // call `amp threads new`
    task := Task{ID:id, Repo:body.Repo, Branch:branch,
                 ThreadID:thread, Prompt:body.Prompt, Status:"queued"}
    db.Create(&task)
    c.JSON(201, gin.H{"id": id, "branch": branch})
}
```

### 12.2 CLI `start` (Go Cobra)

```go
func startCmd(cmd *cobra.Command, args []string) {
    body := map[string]string{
        "repo":  repoFlag,
        "prompt": taskFlag,
    }
    resp, _ := http.Post(apiURL+"/tasks", "application/json", toJSON(body))
    var out struct{ID, Branch string}
    json.NewDecoder(resp.Body).Decode(&out)
    fmt.Printf("📋 Task %s created on branch %s\n", out.ID, out.Branch)
}
```

### 12.3 Wait for CI (helper)

```bash
waitForCI() {
  local branch=$1 sha=$2
  while true; do
    run=$(gh run list --branch "$branch" --limit 1 --json conclusion,headSha,id \
           | jq -r '.[0]')
    [[ $(jq -r '.headSha' <<<"$run") != "$sha" ]] && sleep 10 && continue
    concl=$(jq -r '.conclusion' <<<"$run")
    id=$(jq -r '.id' <<<"$run")
    [[ $concl == "null" ]] && sleep 15 && continue
    echo "$id,$concl"; return
  done
}
```
