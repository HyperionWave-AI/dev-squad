# Hyper Init with AI Provider Configuration

## Overview

The `hyper init` command now supports **direct AI provider configuration** with automatic API token validation. This ensures your credentials work before creating the project files.

## Basic Usage

### Default (Ollama - No Validation Required)

```bash
hyper init
```

Creates a project using **Ollama** for embeddings (free, local, GPU-accelerated).

---

### With OpenAI

```bash
hyper init \
  -provider openai \
  -model gpt-4 \
  -token sk-proj-your-key-here
```

**What it does:**
1. ✅ Validates your OpenAI API key
2. 📝 Creates docker-compose.yml (MongoDB + Qdrant only, no Ollama)
3. 📝 Creates .env.hyper with OpenAI configuration
4. 📝 Creates HYPER_README.md

---

### With Anthropic (Claude)

```bash
hyper init \
  -provider anthropic \
  -model claude-sonnet-4-20250514 \
  -token sk-ant-your-key-here
```

**What it does:**
1. ✅ Validates your Anthropic API key
2. 📝 Creates docker-compose.yml (MongoDB + Qdrant + Ollama for embeddings)
3. 📝 Creates .env.hyper with Anthropic configuration
4. 📝 Creates HYPER_README.md

**Note:** Anthropic doesn't provide embeddings, so Ollama is still used for that.

---

### With Voyage AI

```bash
hyper init \
  -provider voyage \
  -model voyage-3 \
  -token pa-your-voyage-key-here
```

**What it does:**
1. ✅ Validates your Voyage AI API key
2. 📝 Creates docker-compose.yml (MongoDB + Qdrant only, no Ollama)
3. 📝 Creates .env.hyper with Voyage configuration (AI + embeddings)
4. 📝 Creates HYPER_README.md

**Note:** Voyage provides both AI and embeddings, so Ollama is not needed.

---

## Command-Line Flags

| Flag | Required | Description | Example |
|------|----------|-------------|---------|
| **-provider** | No | AI provider name | `openai`, `anthropic`, `voyage`, `ollama` |
| **-model** | No | Model name | `gpt-4`, `claude-sonnet-4`, `voyage-3` |
| **-token** | Yes* | API key/token | `sk-proj-...`, `sk-ant-...`, `pa-...` |
| **-api-url** | No | Custom API endpoint | `https://api.openai.com/v1` |

\* Required for cloud providers (openai, anthropic, voyage). Not required for ollama.

---

## Validation

The `hyper init` command **validates your API credentials** before creating any files:

### ✅ What Gets Validated

| Provider | Validation Method |
|----------|-------------------|
| **OpenAI** | GET `/v1/models` - checks if key can list models |
| **Anthropic** | POST `/v1/messages` - minimal test request |
| **Voyage AI** | POST `/v1/embeddings` - test embedding generation |
| **Ollama** | No validation (local service) |

### ❌ Common Validation Errors

**Invalid API Key:**
```bash
Error: provider validation failed: invalid API key (HTTP 401)
```
**Solution:** Check your API key and try again.

**Missing Token:**
```bash
Error: provider validation failed: OpenAI API token is required (use -token flag)
```
**Solution:** Add `-token your-key-here` to the command.

**Network Error:**
```bash
Error: provider validation failed: failed to connect to OpenAI API: connection timeout
```
**Solution:** Check your internet connection and firewall settings.

---

## Generated Configuration

### Example 1: OpenAI

**Command:**
```bash
hyper init -provider openai -model gpt-4 -token sk-proj-abc123...
```

**Generated .env.hyper:**
```bash
# ... standard config ...

# EMBEDDINGS (Commented out - using OpenAI)
# EMBEDDING=ollama  # Using OpenAI for AI
# OLLAMA_URL=http://localhost:7335
# OLLAMA_MODEL=nomic-embed-text

# ==========================================
# AI PROVIDER CONFIGURATION
# ==========================================
AI_PROVIDER=openai
OPENAI_API_KEY=sk-proj-abc123...
AI_MODEL=gpt-4
```

---

### Example 2: Anthropic

**Command:**
```bash
hyper init -provider anthropic -model claude-sonnet-4-20250514 -token sk-ant-xyz789...
```

**Generated .env.hyper:**
```bash
# ... standard config ...

# EMBEDDINGS (Ollama still used for embeddings)
EMBEDDING=ollama
OLLAMA_URL=http://localhost:7335
OLLAMA_MODEL=nomic-embed-text

# ==========================================
# AI PROVIDER CONFIGURATION
# ==========================================
AI_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-xyz789...
AI_MODEL=claude-sonnet-4-20250514

# Note: Still using Ollama for embeddings (Anthropic doesn't provide embeddings)
```

---

### Example 3: Voyage AI

**Command:**
```bash
hyper init -provider voyage -model voyage-3 -token pa-abc123...
```

**Generated .env.hyper:**
```bash
# ... standard config ...

# EMBEDDINGS (Using Voyage)
EMBEDDING=voyage
# OLLAMA_URL=http://localhost:7335
# OLLAMA_MODEL=nomic-embed-text

# Alternative: Voyage AI (Cloud - Production)
# Voyage AI configured above
VOYAGE_MODEL=voyage-3

# ==========================================
# AI PROVIDER CONFIGURATION
# ==========================================
AI_PROVIDER=voyage
VOYAGE_API_KEY=pa-abc123...
```

---

## Custom API URLs

You can override the default API endpoints:

### OpenAI Compatible API

```bash
hyper init \
  -provider openai \
  -model gpt-4 \
  -token sk-your-key \
  -api-url https://your-openai-compatible-api.com/v1
```

### Self-Hosted Anthropic

```bash
hyper init \
  -provider anthropic \
  -model claude-sonnet-4 \
  -token sk-ant-your-key \
  -api-url https://your-anthropic-endpoint.com/v1
```

---

## Default Models

If you don't specify `-model`, these defaults are used:

| Provider | Default Model |
|----------|---------------|
| **OpenAI** | `gpt-4` |
| **Anthropic** | `claude-sonnet-4-20250514` |
| **Voyage AI** | `voyage-3` |
| **Ollama** | `nomic-embed-text` |

---

## Complete Examples

### Example 1: OpenAI for Production

```bash
# 1. Create project with OpenAI
mkdir my-openai-project && cd my-openai-project
hyper init -provider openai -model gpt-4 -token sk-proj-your-key

# 2. Start only MongoDB + Qdrant (no Ollama needed)
docker compose up -d mongodb qdrant

# 3. Run Hyper
hyper --mode=http

# 4. Access UI
open http://localhost:7095
```

---

### Example 2: Anthropic for AI + Ollama for Embeddings

```bash
# 1. Create project with Anthropic
mkdir my-claude-project && cd my-claude-project
hyper init -provider anthropic -model claude-sonnet-4 -token sk-ant-your-key

# 2. Start all services (includes Ollama for embeddings)
docker compose up -d

# 3. Wait for Ollama model download
docker compose logs -f ollama-pull

# 4. Run Hyper
hyper --mode=http
```

---

### Example 3: Voyage AI for Everything

```bash
# 1. Create project with Voyage
mkdir my-voyage-project && cd my-voyage-project
hyper init -provider voyage -model voyage-3 -token pa-your-key

# 2. Start only MongoDB + Qdrant (Voyage does both AI and embeddings)
docker compose up -d mongodb qdrant

# 3. Run Hyper
hyper --mode=http
```

---

## Validation Output

### Successful Validation

```bash
$ hyper init -provider openai -model gpt-4 -token sk-proj-abc123...

🔐 Validating openai API credentials...
✅ OpenAI API key validated successfully
📝 Creating docker-compose.yml...
✅ docker-compose.yml created
📝 Creating .env.hyper...
✅ .env.hyper created
📝 Creating HYPER_README.md...
✅ HYPER_README.md created

🎉 Hyperion initialized successfully!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📁 Location: current-directory

📝 Files created:
   ✓ docker-compose.yml  - MongoDB + Qdrant + Ollama
   ✓ .env.hyper          - Configuration file
   ✓ HYPER_README.md     - Setup instructions

🚀 Next steps:
   ...

🔧 Provider Configuration:
   Provider: openai
   Model: gpt-4
   Token: sk-proj-...c123
```

---

### Failed Validation

```bash
$ hyper init -provider openai -model gpt-4 -token sk-invalid

🔐 Validating openai API credentials...
Error: provider validation failed: invalid API key (HTTP 401)
```

**No files are created when validation fails!**

---

## Security

### Token Storage

- API tokens are stored in `.env.hyper` (plain text)
- **Important:** Add `.env.hyper` to `.gitignore`
- Never commit API keys to version control

### Token Validation

- Tokens are validated with **minimal API calls** (cost: ~$0.0001)
- Validation timeout: 10-15 seconds
- Network errors are clearly reported

### Token Display

When initialization succeeds, only the first 8 and last 4 characters are shown:

```
Token: sk-proj-...c123
```

---

## Troubleshooting

### Q: Validation is slow

**A:** First-time validation may take 10-15 seconds due to API cold start. Subsequent calls are faster.

---

### Q: Validation fails but key works elsewhere

**A:** Check these:
1. API key is correct (no spaces, complete)
2. API endpoint is reachable (firewall, VPN)
3. Provider service is online (check status page)

---

### Q: Want to skip validation?

**A:** Don't use `-provider` flag. Edit `.env.hyper` manually after `hyper init`.

---

### Q: Want to change provider later?

**A:** Just edit `.env.hyper` and change the `AI_PROVIDER` and related fields.

---

## Comparison

| Setup Method | Validation | Speed | Flexibility |
|--------------|-----------|-------|-------------|
| **hyper init** (no provider) | ❌ None | ⚡ Fast (instant) | ✅ Full (edit later) |
| **hyper init -provider** | ✅ Yes | 🐢 Slower (10-15s) | ✅ Full (edit later) |
| **Manual .env.hyper edit** | ❌ None | ⚡ Fast | ✅ Full |

---

## Summary

### ✅ Benefits

1. **Instant validation** - Know your credentials work before setup
2. **Pre-configured** - No manual .env.hyper editing needed
3. **Flexible** - Support for OpenAI, Anthropic, Voyage, Ollama
4. **Safe** - No files created if validation fails
5. **Custom URLs** - Support for self-hosted endpoints

### 📝 Supported Providers

- ✅ **OpenAI** (gpt-4, gpt-3.5-turbo, etc.)
- ✅ **Anthropic** (claude-sonnet-4, claude-opus, etc.)
- ✅ **Voyage AI** (voyage-3, voyage-2, etc.)
- ✅ **Ollama** (local, no validation)

### 🔧 Required Flags

- **All providers**: `-provider` and `-model` (or use defaults)
- **Cloud providers**: `-token` (mandatory, validated)
- **Ollama**: No flags required (default)

---

**Version:** 1.0.0
**Last Updated:** 2025-01-06
**Command:** `hyper init -provider <name> -token <key>`
