# Releasing katich

This document describes how to create releases for the katich CLI tool.

## Release Process

### Automatic Release (Recommended)

Releases are automatically created when you push a git tag that starts with `v`:

1. **Create a git tag:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build binaries for Windows, Linux, and macOS (Intel and Apple Silicon)
   - Create archives (ZIP for Windows, tar.gz for Linux/Mac)
   - Generate SHA256 checksums
   - Create a GitHub release with all artifacts
   - Generate release notes from git commits

### Manual Release

You can also trigger a release manually:

1. Go to **Actions** → **Release** workflow in GitHub
2. Click **Run workflow**
3. Select the branch (usually `main`)
4. Optionally provide a tag name (e.g., `v1.0.0`)
5. Click **Run workflow**

The workflow will build and release using the specified tag or the latest commit.

## Versioning

We follow [Semantic Versioning](https://semver.org/):
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible

### Pre-releases

Tags with `-alpha`, `-beta`, or `-rc` suffixes are automatically marked as pre-releases:
- `v1.0.0-alpha.1`
- `v1.0.0-beta.1`
- `v1.0.0-rc.1`

## Release Artifacts

Each release includes:

### Binaries
- **Linux**: `katich-linux-amd64`, `katich-linux-arm64`, `katich-linux-arm`
- **macOS**: 
  - `katich-darwin-amd64` (Intel Macs)
  - `katich-darwin-arm64` (Apple Silicon - M1, M2, M3, etc.)

### Archives
- **Linux**: `katich_linux_amd64.tar.gz`, `katich_linux_arm64.tar.gz`, `katich_linux_armv6.tar.gz`, `katich_linux_armv7.tar.gz`
- **macOS**: `katich_darwin_amd64.tar.gz`, `katich_darwin_arm64.tar.gz`

### Checksums
- `katich_{version}_checksums.txt` - SHA256 checksums for all archives

## Downloading Releases

Users can download releases from:
- **GitHub Releases**: https://github.com/kodehash/katichai/releases
- **Latest Release**: https://github.com/kodehash/katichai/releases/latest

### Installation Examples

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_arm64.tar.gz | tar -xz

# Remove macOS quarantine attribute (required for unsigned binaries)
xattr -d com.apple.quarantine katich-darwin-arm64 2>/dev/null || true

sudo mv katich-darwin-arm64 /usr/local/bin/katich
```

**macOS (Intel):**
```bash
curl -L https://github.com/kodehash/katichai/releases/latest/download/katich_darwin_amd64.tar.gz | tar -xz

# Remove macOS quarantine attribute (required for unsigned binaries)
xattr -d com.apple.quarantine katich-darwin-amd64 2>/dev/null || true

sudo mv katich-darwin-amd64 /usr/local/bin/katich
```

**Note for macOS users:** If you see a "cannot be opened" security warning:
1. **Option 1 (Recommended):** Run the `xattr -d com.apple.quarantine` command above before moving the binary
2. **Option 2:** Right-click the binary in Finder → Select "Open" → Click "Open" in the security dialog (first time only)
3. **Option 3:** Go to System Settings → Privacy & Security → Scroll down and click "Open Anyway" next to the blocked app

**Linux:**
```bash
curl -L https://github.com/katichai/katich/releases/latest/download/katich_linux_amd64.tar.gz | tar -xz
sudo mv katich-linux-amd64 /usr/local/bin/katich
```

## Verifying Releases

All releases include SHA256 checksums. To verify a download:

```bash
# Download the checksums file
curl -L https://github.com/katichai/katich/releases/latest/download/katich_{version}_checksums.txt

# Verify your download
shasum -a 256 katich_darwin_arm64.tar.gz
# Compare with the checksum in the file
```

## Release Notes

Release notes are automatically generated from git commits since the last release. The format includes:
- Features
- Bug fixes
- Breaking changes
- Documentation updates

You can edit the release notes on GitHub after the release is created.

## Troubleshooting

### Release workflow fails

1. Check GitHub Actions logs for errors
2. Ensure the tag format is correct (`v*`)
3. Verify GoReleaser configuration is valid
4. Check that `GITHUB_TOKEN` has write permissions

### Binaries not building for a platform

1. Verify the platform is listed in `.goreleaser.yml`
2. Check Go version compatibility
3. Review build logs for specific errors

## Configuration

Release configuration is in:
- `.goreleaser.yml` - GoReleaser build and release settings
- `.github/workflows/release.yml` - GitHub Actions workflow

