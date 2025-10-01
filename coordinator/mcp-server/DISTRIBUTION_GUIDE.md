# Hyperion Coordinator MCP - Distribution Guide

This guide explains how to make the MCP server easy for others to install.

---

## 📊 Installation Options Summary

| Method | User Effort | Maintenance | Best For |
|--------|-------------|-------------|----------|
| **NPM Package** | One command | Auto-updates | 95% of users |
| **One-Click Installer** | Run script | Manual updates | Non-technical users |
| **Pre-built Binaries** | Download + configure | Manual updates | Air-gapped systems |
| **Docker Image** | Docker pull | Docker updates | Containerized environments |
| **Build from Source** | Clone + build | Git pull | Developers |

---

## 🚀 Recommended Approach: NPM Package

### Why NPM?

✅ **Familiar**: Users already have npm for Claude Code
✅ **Simple**: One command install
✅ **Automatic**: Auto-configures Claude Code
✅ **Updates**: `npm update` keeps it current
✅ **Cross-platform**: Works on macOS, Linux, Windows

### Implementation Steps

#### 1. Prepare Package

Files already created:
- ✅ `package.json` - Package metadata and scripts
- ✅ `scripts/build-binary.js` - Builds Go binary on install
- ✅ `scripts/configure-claude.js` - Auto-configures Claude Code

#### 2. Test Locally

```bash
cd coordinator/mcp-server

# Test local installation
npm pack
npm install -g ./hyperion-coordinator-mcp-1.0.0.tgz

# Verify it works
hyperion-coordinator-mcp --version

# Test in Claude Code
# (restart Claude Code and test MCP tools)

# Cleanup
npm uninstall -g @hyperion/coordinator-mcp
```

#### 3. Publish to npm

```bash
# Login to npm (one-time setup)
npm login

# Publish package
npm publish --access public

# Future updates
npm version patch  # 1.0.0 -> 1.0.1
npm publish
```

#### 4. User Installation

Users simply run:

```bash
npm install -g @hyperion/coordinator-mcp
```

That's it! Binary builds, Claude Code configures automatically.

---

## 🛠️ Alternative: One-Click Installer Script

For users who prefer not to use npm.

### What We've Created

✅ `install.sh` - Interactive installer script

### Features

- Detects platform automatically (macOS/Linux/Windows)
- Offers 3 installation methods:
  1. npm (if available)
  2. Download pre-built binary
  3. Build from source (if Go installed)
- Auto-configures Claude Code
- Beautiful colored output
- Error handling and fallbacks

### Usage

```bash
# Give users this one command
curl -fsSL https://raw.githubusercontent.com/yourorg/hyperion-coordinator-mcp/main/install.sh | bash
```

Or download and run:

```bash
wget https://raw.githubusercontent.com/yourorg/hyperion-coordinator-mcp/main/install.sh
chmod +x install.sh
./install.sh
```

---

## 📦 Pre-built Binaries (GitHub Releases)

### Setup CI/CD for Binary Builds

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - os: darwin
            arch: amd64
          - os: darwin
            arch: arm64
          - os: linux
            arch: amd64
          - os: linux
            arch: arm64
          - os: windows
            arch: amd64

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Build binary
        env:
          GOOS: ${{ matrix.os }}
          GOARCH: ${{ matrix.arch }}
        run: |
          go build -o hyperion-coordinator-mcp-${{ matrix.os }}-${{ matrix.arch }} main.go

      - name: Upload Release Asset
        uses: actions/upload-release-asset@v1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          upload_url: ${{ steps.create_release.outputs.upload_url }}
          asset_path: ./hyperion-coordinator-mcp-${{ matrix.os }}-${{ matrix.arch }}
          asset_name: hyperion-coordinator-mcp-${{ matrix.os }}-${{ matrix.arch }}
          asset_content_type: application/octet-stream
```

### Users Download Binary

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yourorg/hyperion-coordinator-mcp/releases/latest/download/hyperion-coordinator-mcp-darwin-arm64 -o hyperion-coordinator-mcp
chmod +x hyperion-coordinator-mcp
sudo mv hyperion-coordinator-mcp /usr/local/bin/
```

---

## 🐳 Docker Distribution

### What We've Created

✅ `Dockerfile` - Multi-stage build for minimal image

### Build and Publish

```bash
# Build image
docker build -t hyperion/coordinator-mcp:latest .

# Tag for registry
docker tag hyperion/coordinator-mcp:latest ghcr.io/yourorg/coordinator-mcp:latest

# Push to GitHub Container Registry
docker push ghcr.io/yourorg/coordinator-mcp:latest
```

### Users Run Container

```bash
docker run -d \
  --name hyperion-coordinator-mcp \
  -e MONGODB_URI="your_uri" \
  ghcr.io/yourorg/coordinator-mcp:latest
```

---

## 📝 Documentation for Users

### What We've Created

✅ `INSTALLATION.md` - Comprehensive installation guide covering all methods

### Key Sections

1. **Quick Install** - npm one-liner (recommended)
2. **Manual Options** - All alternative methods
3. **Configuration** - Environment variables
4. **Testing** - Verification steps
5. **Troubleshooting** - Common issues

### Distribution

Include installation link everywhere:

- **GitHub README**:
  ```markdown
  ## Installation

  ```bash
  npm install -g @hyperion/coordinator-mcp
  ```

  See [INSTALLATION.md](./INSTALLATION.md) for other options.
  ```

- **npm Package Description**:
  ```json
  "description": "MCP server for AI agent coordination. Install: npm install -g @hyperion/coordinator-mcp"
  ```

- **Social Media**:
  > Install Hyperion Coordinator MCP in one command:
  > `npm install -g @hyperion/coordinator-mcp`

---

## 🎯 Marketing the Package

### npm Package Discoverability

1. **Good Package Name**: `@hyperion/coordinator-mcp` (scoped, descriptive)
2. **Keywords**: In `package.json`:
   ```json
   "keywords": ["mcp", "model-context-protocol", "ai-agents", "task-coordination"]
   ```
3. **README Badge**:
   ```markdown
   ![npm version](https://img.shields.io/npm/v/@hyperion/coordinator-mcp.svg)
   ![npm downloads](https://img.shields.io/npm/dm/@hyperion/coordinator-mcp.svg)
   ```

### GitHub Repository

1. **Clear README** with installation front and center
2. **Releases Page** with pre-built binaries
3. **Wiki** with detailed guides
4. **Discussions** for Q&A
5. **Topics**: Add tags like `mcp`, `claude-code`, `ai-agents`

### Community Engagement

1. **MCP Registry**: Submit to official MCP registry (if available)
2. **Blog Post**: "How to Install Hyperion Coordinator MCP"
3. **Video Tutorial**: YouTube walkthrough
4. **Reddit/HN**: "Show HN: One-command MCP server installation"

---

## 🔄 Update Strategy

### For npm Users

Automatic update reminders:

```json
// In package.json
"scripts": {
  "postinstall": "node -e \"console.log('\\nTip: Run \\`npm update -g @hyperion/coordinator-mcp\\` to get the latest version!\\n')\""
}
```

### For Binary Users

Include update check in binary:

```go
// In main.go
func checkForUpdates() {
    latestVersion := fetchLatestVersion()
    if latestVersion > currentVersion {
        fmt.Printf("⚠️  Update available: %s → %s\n", currentVersion, latestVersion)
        fmt.Println("Run: npm update -g @hyperion/coordinator-mcp")
    }
}
```

### Versioning Strategy

Follow semantic versioning:

- **Major** (1.0.0 → 2.0.0): Breaking changes
- **Minor** (1.0.0 → 1.1.0): New features
- **Patch** (1.0.0 → 1.0.1): Bug fixes

---

## 📊 Analytics and Feedback

### Track Adoption

- **npm downloads**: https://npmtrends.com/@hyperion/coordinator-mcp
- **GitHub stars/forks**: Repository insights
- **GitHub Container Registry pulls**: Package insights

### Gather Feedback

1. **GitHub Issues**: Bug reports and feature requests
2. **GitHub Discussions**: Q&A and community help
3. **npm Survey**: Post-install questionnaire
4. **Usage Analytics**: Telemetry (opt-in only!)

---

## ✅ Pre-Launch Checklist

Before publishing:

- [ ] Test npm package installation on all platforms (macOS, Linux, Windows)
- [ ] Verify auto-configuration works
- [ ] Test pre-built binaries on all platforms
- [ ] Ensure Docker image builds and runs
- [ ] Write comprehensive README
- [ ] Create INSTALLATION.md with troubleshooting
- [ ] Set up GitHub releases with CI/CD
- [ ] Create issue templates
- [ ] Write CONTRIBUTING.md for open-source contributors
- [ ] Add LICENSE file
- [ ] Create CHANGELOG.md
- [ ] Test update process (npm update)
- [ ] Verify uninstall cleans up properly
- [ ] Create quick start video (optional but helpful)

---

## 🚀 Launch Plan

### Week 1: Soft Launch

1. Publish to npm as `@hyperion/coordinator-mcp@0.9.0` (beta)
2. Share with small group of testers
3. Gather feedback and fix issues

### Week 2: Official Launch

1. Update to `@hyperion/coordinator-mcp@1.0.0`
2. Publish GitHub release with binaries
3. Update documentation
4. Announce on:
   - GitHub Discussions
   - Reddit (r/MachineLearning, r/ClaudeAI)
   - Twitter/X
   - LinkedIn

### Week 3+: Growth

1. Monitor GitHub issues
2. Respond to questions in Discussions
3. Publish blog posts / tutorials
4. Add to MCP registry (if available)
5. Create video walkthrough

---

## 🎁 Bonus: Web Installer

Create https://install.hyperion.dev with:

```bash
# One-line install
curl -fsSL https://install.hyperion.dev/coordinator-mcp | bash
```

This URL redirects to the install.sh script on GitHub.

---

## 📞 Support Channels

Set up:

1. **GitHub Issues**: Bug tracking
2. **GitHub Discussions**: Community Q&A
3. **Discord/Slack**: Real-time help (optional)
4. **Email**: support@hyperion.dev (optional)
5. **Documentation Site**: docs.hyperion.dev (optional)

---

## 🎯 Success Metrics

Track these to measure adoption:

- npm downloads per week
- GitHub stars
- GitHub issues (engagement)
- Discord/community members
- Twitter mentions
- Blog post views

**Goal**: 1000+ downloads in first month

---

**Ready to launch?** Follow the Pre-Launch Checklist and you're good to go! 🚀
