# OAuth MCP Proxy - Implementation Log

> **Purpose:** Strict record of implementation progress, decisions, and changes.

**Plan Reference:** [docs/plan.md](plan.md)

---

## Phase 0: Repository Setup

**Status:** 🔄 In Progress

**Started:** 2025-10-17

### Tasks

- [ ] Initialize go.mod (`go mod init github.com/tuannvm/oauth-mcp-proxy`)
- [ ] Copy all `.go` files from `../mcp-trino/internal/oauth/`
- [ ] Set up .gitignore, LICENSE (MIT)
- [ ] Update imports from `internal/oauth` → root package
- [ ] Commit: "Initial extraction from mcp-trino"

### Implementation Notes

*No entries yet. Record all decisions, blockers, and changes here as work progresses.*

---

## Phase 1: Make It Compile

**Status:** ⏳ Not Started

### Tasks

- [ ] Remove Trino-specific imports (`internal/config`)
- [ ] Update imports from `internal/oauth` → root
- [ ] Replace Trino config with generic Config
- [ ] Fix compilation errors (minimal changes)

**Success:** `go build ./...` works

### Implementation Notes

*Record decisions, blockers, changes here during Phase 1.*

---

## Phase 1.5: Critical Architecture Fixes

**Status:** ⏳ Not Started

### Tasks

- [ ] Fix global token cache → Instance-scoped (move to internal/cache/)
- [ ] Add Logger interface → No hardcoded log.Printf()
- [ ] Add Config.Validate() method → Fail fast on invalid config

### Implementation Notes

*Record architectural changes and decisions here.*

---

## Phase 2: Package Structure

**Status:** ⏳ Not Started

### Tasks

- [ ] Move providers to provider/ package
- [ ] Keep handlers in ROOT (need Server internals)
- [ ] Move cache to internal/cache/
- [ ] Update imports

### Implementation Notes

*TBD*

---

## Phase 3: Simple API Implementation

**Status:** ⏳ Not Started

### Tasks

- [ ] Implement oauth.EnableOAuth() in ROOT package
- [ ] Auto-detect mode with validation
- [ ] Test both modes

### Implementation Notes

*TBD*

---

## Phase 4: OAuth-Only Tests

**Status:** ⏳ Not Started

### Implementation Notes

*TBD*

---

## Phase 5: Documentation

**Status:** ⏳ Not Started

### Implementation Notes

*TBD*

---

## Phase 6: Migration Validation

**Status:** ⏳ Not Started

### Implementation Notes

*TBD*

---

## Decisions Log

| Date | Phase | Decision | Rationale |
|------|-------|----------|-----------|
| 2025-10-17 | Planning | Adopted "Extract then Fix" strategy | Lower risk, no mcp-trino changes during dev |
| 2025-10-17 | Planning | Added metrics as P0 issue #12 | Gemini 2.5 Pro feedback: standalone needs observability |
| 2025-10-17 | Planning | MCP adapter interface design moved to Phase 1 | Build core against predefined contract |
| 2025-10-17 | Planning | Adopted structured package layout | provider/ and handler/ subpackages for better organization |
| 2025-10-17 | Planning | Split embedded vs standalone mode | Focus v0.1.0 on embedded only, defer standalone to v0.2.0 |
| 2025-10-17 | Planning | Cleaned up plan.md (Option B) | Replaced with embedded-only version, backed up old to plan-full-original.md |
| 2025-10-17 | Planning | Reordered phases: Work first, refactor later | Phase 4: Tests, Phase 5: Architecture cleanup (was Phase 1) |
| 2025-10-17 | Planning | Deferred Phase 5 (Architecture) to v0.2.0 | Ship working code in v0.1.0, perfect it in v0.2.0 |
| 2025-10-17 | Planning | Adopted Option A (EnableOAuth) as primary API | Simplest possible integration for MCP developers |
| 2025-10-17 | Planning | Auto-detect native vs proxy mode | Based on ClientID presence in config |
| 2025-10-17 | Planning | Library is MCP-only (no adapter pattern) | Name indicates this, no need for abstraction |
| 2025-10-17 | Planning | Handlers stay in root (not handler/) | Need access to Server internals |
| 2025-10-17 | Planning | Added Phase 1.5: Critical Architecture | Fix global state, logging, validation in v0.1.0 |

---

## Blockers & Issues

*Record any blockers or issues encountered during implementation.*

| Date | Phase | Issue | Resolution | Status |
|------|-------|-------|------------|--------|
| - | - | - | - | - |

---

## Document Updates

| Date | Version | Changes |
|------|---------|---------|
| 2025-10-17 | 1.0 | Initial implementation log created |
