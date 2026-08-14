---
schema_version: 1
open_count: 2
waived_count: 0
fixed_count: 0
total_count: 2
last_updated: 2026-08-14T19:25:08.710Z
---

# Broken Windows Ledger

> Cross-phase defect register. `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 04 | unrun-verify | cmd/alturd/main.go |  | UAT test 2 (git difftool -t alturd <file> real-terminal launch, DIFFTOOL-01/G-04-2) deferred to phase re-verification; requires real interactive terminal, cannot be run by executor agent. | open |  | 2026-08-06T10:06:51.665Z |  |
| 2 | 04.1 | unrun-verify | cmd/alturd/loaddifftoolfiles_internal_test.go |  | go test -race ./... could not run in this execution environment (CGO_ENABLED=0, no gcc/cc on PATH, no root to install one); go test ./... (without -race) run instead and is fully green — plan verification block requires -race | open |  | 2026-08-14T19:25:08.710Z |  |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "04",
    "file": "cmd/alturd/main.go",
    "line": null,
    "description": "UAT test 2 (git difftool -t alturd <file> real-terminal launch, DIFFTOOL-01/G-04-2) deferred to phase re-verification; requires real interactive terminal, cannot be run by executor agent.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-06T10:06:51.665Z",
    "resolved_at": null
  },
  {
    "id": 2,
    "kind": "unrun-verify",
    "phase": "04.1",
    "file": "cmd/alturd/loaddifftoolfiles_internal_test.go",
    "line": null,
    "description": "go test -race ./... could not run in this execution environment (CGO_ENABLED=0, no gcc/cc on PATH, no root to install one); go test ./... (without -race) run instead and is fully green — plan verification block requires -race",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-14T19:25:08.710Z",
    "resolved_at": null
  }
]
````
