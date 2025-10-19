# Release Process

Guide for maintainers on releasing new versions of oauth-mcp-proxy.

---

## Publishing to pkg.go.dev

### How Go Module Publishing Works

1. **Automatic Indexing:**
   - Push code to GitHub
   - Create git tag (e.g., `v0.1.0`)
   - First visit to `pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy` triggers indexing
   - Documentation appears automatically

2. **No Registration Required:**
   - pkg.go.dev indexes all public Go modules on GitHub
   - Just need valid `go.mod` and git tag

3. **Update Frequency:**
   - New versions indexed on first request
   - Usually within minutes of tag push
   - Can force refresh by requesting the version

---

## Release Workflow

### Prerequisites

- [ ] All tests passing (`go test ./...`)
- [ ] Phase 6 complete (mcp-trino migration validated)
- [ ] CHANGELOG.md updated
- [ ] Documentation reviewed
- [ ] Examples tested

### Automated Release (Recommended)

1. **Go to GitHub Actions**
   - Navigate to Actions tab
   - Select "Release" workflow
   - Click "Run workflow"

2. **Choose Version Bump:**
   - **patch** - Bug fixes (0.1.0 → 0.1.1)
   - **minor** - New features (0.1.0 → 0.2.0)
   - **major** - Breaking changes (0.1.0 → 1.0.0)

3. **Workflow Automatically:**
   - Runs tests
   - Bumps version and creates tag
   - Generates changelog
   - Creates GitHub Release
   - Requests pkg.go.dev indexing

### Manual Release

```bash
# 1. Ensure clean state
git status
go test ./...

# 2. Update CHANGELOG.md
# Add release notes for the version

# 3. Commit changelog
git add CHANGELOG.md
git commit -m "chore: update changelog for v0.1.0"
git push

# 4. Create and push tag
git tag -a v0.1.0 -m "Release v0.1.0: OAuth library for MCP servers"
git push origin v0.1.0

# 5. Create GitHub Release (using gh CLI)
gh release create v0.1.0 \
  --title "v0.1.0" \
  --generate-notes

# 6. Request pkg.go.dev indexing
curl https://proxy.golang.org/github.com/tuannvm/oauth-mcp-proxy/@v/v0.1.0.info

# 7. Verify on pkg.go.dev
open https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy@v0.1.0
```

---

## Versioning Strategy

### Semantic Versioning

**v0.1.0 (Current):**
- Embedded mode library
- 4 providers (HMAC, Okta, Google, Azure)
- Native and proxy modes

**v0.2.0 (Planned):**
- Standalone proxy service
- Additional architecture improvements
- Breaking changes OK (still v0.x)

**v1.0.0 (Future):**
- Stable API
- No breaking changes in v1.x releases

### Version Bump Guidelines

**Patch** (0.1.0 → 0.1.1):
- Bug fixes
- Documentation updates
- Security patches
- No API changes

**Minor** (0.1.0 → 0.2.0):
- New features
- New providers
- API additions (backward compatible)
- Can include breaking changes in v0.x

**Major** (0.9.0 → 1.0.0):
- API stability commitment
- Breaking changes after v1.0
- Major architecture changes

---

## Release Checklist

### Pre-Release

- [ ] All Phase requirements completed
- [ ] Tests passing (`go test -race ./...`)
- [ ] Examples build successfully
- [ ] Documentation up to date
- [ ] CHANGELOG.md updated
- [ ] No pending security issues
- [ ] GoDoc comments complete

### Release

- [ ] Version tag created (vX.Y.Z)
- [ ] Tag pushed to GitHub
- [ ] GitHub Release created
- [ ] Release notes generated
- [ ] pkg.go.dev indexing requested

### Post-Release

- [ ] Verify pkg.go.dev documentation
- [ ] Test installation: `go get github.com/tuannvm/oauth-mcp-proxy@vX.Y.Z`
- [ ] Update README badges if needed
- [ ] Announce on relevant channels
- [ ] Update mcp-trino dependency (after v0.1.0)

---

## pkg.go.dev Tips

### Triggering Indexing

After pushing a tag:

```bash
# Request specific version
curl https://proxy.golang.org/github.com/tuannvm/oauth-mcp-proxy/@v/v0.1.0.info

# Request latest
curl https://proxy.golang.org/github.com/tuannvm/oauth-mcp-proxy/@latest

# Or just visit the page (triggers indexing)
open https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy
```

### Documentation Quality

pkg.go.dev shows:
- ✅ Package overview (from package comment)
- ✅ All public APIs with GoDoc
- ✅ Examples (from `_test.go` files with Example functions)
- ✅ Source code links

**Verify:**
1. All public types/functions have comments
2. Comments start with type/function name
3. Examples use standard Go example format

---

## Testing Installation

After release:

```bash
# Create test directory
mkdir /tmp/test-oauth-mcp-proxy
cd /tmp/test-oauth-mcp-proxy

# Initialize module
go mod init test

# Install library
go get github.com/tuannvm/oauth-mcp-proxy@v0.1.0

# Verify
go list -m github.com/tuannvm/oauth-mcp-proxy
# Should show: github.com/tuannvm/oauth-mcp-proxy v0.1.0
```

---

## Rollback

If a release has critical issues:

```bash
# Delete tag locally
git tag -d v0.1.0

# Delete tag on GitHub
git push origin :refs/tags/v0.1.0

# Delete GitHub Release (via gh CLI)
gh release delete v0.1.0 --yes

# Or manually delete on GitHub web UI
```

**Note:** Cannot unpublish from pkg.go.dev once indexed. Instead, release a patch version with fixes.

---

## First Release (v0.1.0)

**After Phase 6 complete:**

```bash
# 1. Final review
go test ./...
go build ./...

# 2. Update CHANGELOG
# Move items from [Unreleased] to [0.1.0]

# 3. Trigger release workflow
# GitHub Actions → Release → Run workflow → patch/minor/major

# 4. Verify release
# Check GitHub Releases page
# Check pkg.go.dev indexing
# Test: go get github.com/tuannvm/oauth-mcp-proxy@v0.1.0

# 5. Announce
# Update mcp-trino to use new library
# Update README status to "Released"
```

---

## Support Policy

**v0.x releases:**
- Latest minor version supported
- Security patches backported to latest only

**v1.x releases (future):**
- Latest minor version fully supported
- Previous minor receives security patches for 6 months
- Breaking changes only in major versions

---

## Questions?

- Review existing releases: [GitHub Releases](https://github.com/tuannvm/oauth-mcp-proxy/releases)
- Check pkg.go.dev status: [pkg.go.dev](https://pkg.go.dev/github.com/tuannvm/oauth-mcp-proxy)
