# OAuth Extraction Strategy - Critical Nuances

## Current Situation

**Source Repo:** `../mcp-trino/internal/oauth/` (~3000 LOC including tests)
**Destination Repo:** `oauth-mcp-proxy` (this repo, currently empty except docs)

## Critical Nuances Detected

### 🔴 Issue 1: Cross-Repo Workflow Complexity

**Current Plan:** "Fix then Extract" (Phase 0 in mcp-trino, then extract)

**Problems:**
- Phase 0 modifies production code in mcp-trino
- Requires PR approval, testing, deployment in mcp-trino
- Blocks extraction until mcp-trino changes are merged
- Risk: mcp-trino might be actively developed, causing merge conflicts
- Coordination overhead: two repos, two workflows

**Timeline Impact:**
- Phase 0 "5 days" assumes immediate merge approval
- Reality: code review + CI/CD + deployment could take 1-2 weeks
- If mcp-trino has other maintainers, even longer

---

### 🔴 Issue 2: Version Drift Risk

**Scenario:**
1. Week 1: Fix issues in mcp-trino (Phase 0)
2. Week 2: Extract FIXED code to oauth-mcp-proxy
3. Problem: What if mcp-trino code changes between fix and extraction?

**Risk:**
- Extract stale code (before fixes merged)
- Extract wrong version (after other changes)
- Need to keep syncing between repos during Phases 0-2

---

### 🔴 Issue 3: Testing Context Mismatch

**Current Tests Run In:**
- mcp-trino context (imports Trino config, uses internal packages)

**After Extraction:**
- Tests must run standalone (no Trino dependencies)
- But we can't verify this UNTIL after extraction
- Phase 0 fixes might break when extracted

**Example:**
```go
// Current (mcp-trino)
import "github.com/tuannvm/mcp-trino/internal/config"

// After extraction
import "github.com/tuannvm/oauth-mcp-proxy" // different imports
```

---

### 🔴 Issue 4: Production Risk

**Current Plan:** Fix production code (mcp-trino) first

**Risk:**
- Phase 0 changes might break mcp-trino production
- 11 structural changes (globals → instances, middleware registry, etc.)
- High regression risk
- Requires extensive testing in mcp-trino BEFORE extraction

---

### 🔴 Issue 5: Go Module Initialization Gap

**Current State:**
- oauth-mcp-proxy has no `go.mod` yet
- Plan shows Phase 1 (Day 3): "Initialize go.mod"
- But Phase 0 needs to happen FIRST (in different repo)

**Question:** Should we initialize go.mod NOW to start developing?

---

## Recommended Strategy Pivots

### ✅ Option A: "Extract then Fix" (RECOMMENDED)

**Approach:**
1. **Phase 0**: Repository setup (THIS repo)
   - Initialize go.mod
   - Set up CI/CD
   - Copy code AS-IS from mcp-trino

2. **Phase 1**: Fix all 11 issues IN oauth-mcp-proxy
   - No risk to mcp-trino production
   - Can iterate freely
   - Tests run in clean environment

3. **Phase 2-7**: Complete extraction, testing, docs

4. **Phase 8**: Migrate mcp-trino to use new library

**Advantages:**
- ✅ No production risk to mcp-trino
- ✅ Faster (no cross-repo coordination)
- ✅ Tests verified in target environment
- ✅ Can develop independently
- ✅ mcp-trino stays stable during extraction

**Timeline:** Still 30 days, but more predictable

---

### Option B: "Parallel Development"

**Approach:**
1. Fork mcp-trino code NOW (as-is)
2. Develop oauth-mcp-proxy in parallel
3. Once stable, submit fixes BACK to mcp-trino as library import

**Advantages:**
- ✅ Both repos work independently
- ✅ Can test both versions side-by-side

**Disadvantages:**
- ⚠️ Duplicate maintenance during transition
- ⚠️ Need to keep changes in sync

---

### Option C: "Fix then Extract" (ORIGINAL PLAN)

**Only viable if:**
- ✅ You are sole maintainer of mcp-trino
- ✅ mcp-trino is not in production
- ✅ Can merge PRs immediately
- ✅ Can tolerate regression risk

**Otherwise:** Too risky and slow

---

## Additional Nuances to Address

### 1. Go Module Path Strategy

**Question:** Should extracted code use same import paths initially?

```go
// Option 1: Clean break
import "github.com/tuannvm/oauth-mcp-proxy"

// Option 2: Compatibility period
import oauth "github.com/tuannvm/oauth-mcp-proxy"
// Provide type aliases for transition
```

**Recommendation:** Clean break, use migration guide

---

### 2. Test Strategy

**Current:** 1000+ LOC of tests in mcp-trino

**Need to decide:**
- Copy all tests as-is?
- Rewrite tests for new structure?
- Which tests are Trino-specific vs generic OAuth?

**Recommendation:**
- Copy ALL tests initially
- Remove Trino-specific tests
- Add new tests for library use cases

---

### 3. Configuration Loading

**Current:** mcp-trino uses its own config structure

```go
// mcp-trino
type TrinoConfig struct {
    OAuthMode string
    OAuthProvider string
    // ... 20+ fields
}
```

**After extraction:** Need generic Config

**Nuance:** Config validation logic might be Trino-specific

---

### 4. MCP Adapter Timing

**Plan:** Phase 4 (Day 13-15) - MCP Adapter

**Nuance:** Should this be Phase 1?
- Adapter defines the API contract
- Core code designed around adapter interface
- Wrong order: build core first, discover adapter issues later

**Recommendation:** Design adapter interface FIRST (Phase 1)

---

### 5. Version Compatibility

**Missing from plan:**
- What Go version? (mcp-trino uses Go 1.21+)
- Which mcp-go version? (v0.38.0 in mcp-trino)
- Breaking changes between versions?

**Need to specify:**
```go
// go.mod
go 1.21

require (
    github.com/mark3labs/mcp-go v0.38.0
    // ... others
)
```

---

### 6. Documentation Timing

**Current plan:** Phase 6 (Day 21-23)

**Nuance:** Docs should be written DURING development
- Architecture docs guide implementation
- API docs prevent scope creep
- Examples validate design decisions

**Recommendation:** Write docs FIRST (Phase 1), refine later

---

## Revised Timeline with Nuances Addressed

| Phase | Duration | Focus | Location | Risk |
|-------|----------|-------|----------|------|
| **Phase 0** | **3 days** | **Repo setup + copy code** | oauth-mcp-proxy | Low |
| Phase 1 | 2 days | Design interfaces + API | oauth-mcp-proxy | Low |
| Phase 2 | 7 days | Fix 11 P0 issues | oauth-mcp-proxy | Medium |
| Phase 3 | 5 days | Core extraction | oauth-mcp-proxy | Medium |
| Phase 4 | 3 days | Dual mode (embedded + standalone) | oauth-mcp-proxy | Medium |
| Phase 5 | 5 days | Testing + docs | oauth-mcp-proxy | Low |
| Phase 6 | 3 days | mcp-trino migration | mcp-trino | High |
| Phase 7 | 2 days | Validation + release | Both repos | Medium |
| **Total** | **30 days** | **Lower risk, cleaner** | | |

---

## ✅ DECISION CONFIRMED

**Chosen Strategy:** Extract then Fix (Option A)

**Confirmed Approach:**
1. ✅ Copy code from `../mcp-trino/internal/oauth/` to this repo
2. ✅ Fix issues here (no changes to mcp-trino during development)
3. ✅ No releases until extraction complete
4. ✅ Create OAuth-only tests (remove all Trino dependencies)
5. ✅ No timelines/deadlines - work through phases sequentially

**Key Principles:**
- **Isolation:** mcp-trino unchanged during development
- **Focus:** OAuth functionality only, no Trino coupling
- **Quality:** Complete each phase before moving forward
- **Testing:** Build comprehensive OAuth test suite from scratch

---

## Implementation Strategy

### Phase 0: Copy Code
- Initialize go.mod
- Copy all `.go` files from mcp-trino
- Update imports
- First commit

### Phase 1-7: Development
- Fix architecture issues
- Build OAuth library
- Create OAuth-only tests
- Document everything

### Phase 8: Migration
- Update mcp-trino to use new library
- Validate functionality
- Release when ready

---

**Document Version:** 1.1
**Date:** 2025-10-17
**Status:** ✅ Strategy confirmed, ready to execute
