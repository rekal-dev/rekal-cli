# Provenance — artifact → commit → session → intent

Use when the anchor is a specific file, function, line, or commit.

## Funnel (stop when you have the answer)

### 1. Anchor on the artifact

```bash
git log --oneline -15 -- path/to/file.go
git log --oneline -15 -L :FuncName:path.go
git blame -L 120,160 -- path/to/file.go
```

### 2. Commit → sessions

```bash
rekal --commit <sha>
# or
rekal query --index "SELECT DISTINCT cs.session_id FROM checkpoint_sessions cs \
  JOIN checkpoints c ON c.id = cs.checkpoint_id WHERE c.git_sha LIKE '<sha>%'"
```

### 3. Session → intent

```bash
rekal query --session <id> --role human_steering
rekal query --session <id> --role human
rekal query --session <id> --role summary
```

### 4. Emit the why-chain

artifact → commit `<sha>` → session `<id>` → human intent (quoted/steering),
with turn pointers. If no session links to the commit, say so — don't invent.
