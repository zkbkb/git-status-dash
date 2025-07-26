# Fix GitHub Release Pipeline for Go Binaries

## Summary

This PR fixes the critical issue where Go binaries fail to download (404 error) by properly attaching compiled binaries to GitHub releases and adding comprehensive verification stages.

## Problem

Users attempting to install git-status-dash Go binaries receive 404 errors because the release workflow only uploads artifacts temporarily within the workflow context, never attaching them to the actual GitHub release.

```bash
# This fails with 404
curl -L https://github.com/ejfox/git-status-dash/releases/latest/download/git-status-dash-go-linux-amd64
```

## Solution

### 1. Enhanced Release Workflow
- Added `verify-node` and `verify-go` jobs for pre-release validation
- Properly attach binaries using `softprops/action-gh-release`
- Generate SHA256 checksums for all binaries
- Comprehensive error checking and validation

### 2. Local Testing Capability
- Created `test-release.sh` for local release simulation
- Mirrors GitHub Actions workflow locally
- Beautiful colored output with status indicators

### 3. Documentation Updates
- Added workflow status badges
- Clear installation instructions
- Performance comparison data

## Changes

- 📝 Modified `.github/workflows/production_release.yml` - Complete workflow restructure
- ✨ Added `test-release.sh` - Local testing script
- 📚 Created `MODIFICATION_REPORT.md` - Detailed change documentation
- 📊 Created `UPSTREAM_COMPARISON.md` - Comparison with original repository
- 🎯 Created `PULL_REQUEST_TEMPLATE.md` - This PR template

## Testing

### Local Testing
```bash
# Run local release test
./test-release.sh

# Expected: All green checkmarks
```

### CI Testing
```bash
# Create test tag
git tag v0.0.1-test
git push origin v0.0.1-test

# Verify in Actions tab that all 5 jobs pass
# Check release page for binaries
```

## Verification Checklist

- [ ] All 5 workflow jobs pass (verify-node, verify-go, release-nodejs, release-go, create-release)
- [ ] Binaries are attached to GitHub release
- [ ] Download links work (no 404 errors)
- [ ] Checksums are generated and included
- [ ] Local test script passes all checks
- [ ] README badges show correct status

## Breaking Changes

None - Full backward compatibility maintained.

## Screenshots

### Before
```
$ curl -L .../git-status-dash-go-linux-amd64
404: Not Found
```

### After
```
$ curl -L .../git-status-dash-go-linux-amd64
[Binary downloads successfully]
```

## Related Issues

Fixes #2 - Installation script returns 404 for Go binaries

## Additional Notes

- First release after merge should use a test tag to verify
- Ensure `NPM_TOKEN` secret is configured
- Monitor first few releases for any issues

---

**Ready for review!** 🚀