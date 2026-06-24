# Release Workflow Setup

## How It Works

### Automatic (on push to main)
1. **Build job** runs automatically
2. Builds binaries for all platforms
3. Uploads as artifacts (kept for 7 days)
4. Waits at **Release job**

### Manual Approval Required
1. Go to GitHub Actions: https://github.com/akash202k/ctx/actions
2. Click on the running workflow
3. Review the builds
4. Click "Review deployments" → Approve
5. **Release job** runs:
   - Creates tag (if versioned release)
   - Publishes release with binaries

## One-Time Setup: Create Environment

You need to create the `release-approval` environment with required reviewers:

### Steps:
1. Go to: https://github.com/akash202k/ctx/settings/environments
2. Click "New environment"
3. Name: `release-approval`
4. Enable "Required reviewers"
5. Add yourself (or team members) as required reviewers
6. Click "Save protection rules"

### Alternative: Remove approval requirement
If you want automatic releases without approval, remove these lines from `.github/workflows/release.yml`:
```yaml
    environment:
      name: release-approval
```

## Release Types

### Latest Prerelease (automatic after approval)
- Triggered by: push to main
- Version: `latest`
- Tag: none created
- Use: testing, continuous delivery

### Versioned Release (manual trigger)
1. Go to: https://github.com/akash202k/ctx/actions/workflows/release.yml
2. Click "Run workflow"
3. Enter version: `v0.1.3`
4. Approve when build completes
5. Creates: tag + versioned release
