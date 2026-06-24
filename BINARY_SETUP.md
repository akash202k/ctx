# Setup Instructions for Binary Releases

## Current Status

✅ Workflow file created: `.github/workflows/release.yml`  
✅ Smart installer created: `scripts/install.sh` (with binary fallback)  
❌ **Binaries not yet built** (workflow hasn't run)

## To Enable Binary Installation (Without Go):

### Step 1: Push the workflow to GitHub

```bash
cd /Users/akash/akash/repos/ctx

# Add and commit all changes
git add -A
git commit -m "Add release workflow and smart installer with binary fallback"
git push origin main
```

### Step 2: Recreate the v0.1.0 tag

This triggers the workflow to build binaries:

```bash
# Delete old tag locally and remotely
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# Create and push new tag
git tag v0.1.0
git push origin v0.1.0
```

### Step 3: Wait for GitHub Actions (2-3 minutes)

1. Visit: https://github.com/akash202k/ctx/actions
2. Watch the "Release" workflow run
3. It will build 5 binaries:
   - `ctx_0.1.0_linux_amd64.tar.gz`
   - `ctx_0.1.0_linux_arm64.tar.gz`
   - `ctx_0.1.0_darwin_amd64.tar.gz`
   - `ctx_0.1.0_darwin_arm64.tar.gz`
   - `ctx_0.1.0_windows_amd64.zip`

### Step 4: Verify Release

After workflow completes:
1. Visit: https://github.com/akash202k/ctx/releases/tag/v0.1.0
2. You should see the 5 binary files attached
3. Now users can install **without Go**!

### Step 5: Test Installation

```bash
# Uninstall Go (optional, just for testing)
brew uninstall go

# Test the installer
curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.sh | sh

# It should now:
# 1. Detect Go is missing
# 2. Download pre-built binary
# 3. Install successfully!
```

## For Users

### With Go (Always Works):
```bash
curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/setup.sh | sh
```

### Without Go (After Step 4 completes):
```bash
curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.sh | sh
```

## What Was Your Error?

```
tar: Error opening archive: Unrecognized archive format
```

This happened because:
- The installer tried to download: `ctx_0.1.0_darwin_arm64.tar.gz`
- But that file doesn't exist yet on GitHub
- It will exist after you push the workflow and recreate the tag

## After Setup

Once binaries are available, anyone can run:

```bash
curl -fsSL https://raw.githubusercontent.com/akash202k/ctx/main/scripts/install.sh | sh
```

And it will:
1. Check if Go exists
2. If yes → use `go install` (fastest)
3. If no → download pre-built binary (no Go required!)
4. Install successfully either way
