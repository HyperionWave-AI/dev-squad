# Anthropic Claude Sonnet Configuration Guide

## ✅ Configuration Updated

The `.env.hyper` file has been configured to use **Anthropic Claude Sonnet 4.5** as the AI provider.

---

## 🔑 Required: Add Your Anthropic API Key

### Step 1: Get Your Anthropic API Key

1. Go to https://console.anthropic.com/
2. Sign in or create an account
3. Navigate to **API Keys** section
4. Create a new API key or copy an existing one

### Step 2: Update .env.hyper

Open `.env.hyper` and replace the placeholder with your actual API key:

```bash
# Find this line (around line 85):
ANTHROPIC_API_KEY="YOUR_ANTHROPIC_API_KEY_HERE"

# Replace with your actual key:
ANTHROPIC_API_KEY="sk-ant-api03-xxxxxxxxxxxxx"
```

---

## 📋 Current Configuration

### Provider Settings
- **AI Provider:** `anthropic`
- **Model:** `claude-sonnet-4-20250514` (Claude Sonnet 4.5)
- **API Key:** `YOUR_ANTHROPIC_API_KEY_HERE` ⚠️ **NEEDS TO BE UPDATED**

### Model Options (in order of capability)

| Model ID | Description | Context Window | Best For |
|----------|-------------|----------------|----------|
| `claude-sonnet-4-20250514` | Claude Sonnet 4.5 (latest) | 200K tokens | Best overall performance, latest features |
| `claude-3-5-sonnet-20241022` | Claude 3.5 Sonnet | 200K tokens | Very capable, slightly older |
| `claude-3-5-sonnet-20240620` | Claude 3.5 Sonnet (older) | 200K tokens | Stable, well-tested version |
| `claude-3-sonnet-20240229` | Claude 3 Sonnet | 200K tokens | Older but reliable |

To change models, edit this line in `.env.hyper`:
```bash
AI_MODEL="claude-sonnet-4-20250514"
```

---

## 🚀 How to Use

### Option 1: Source the Environment File (Recommended)

```bash
# Load the configuration
source .env.hyper

# Verify it's loaded
echo $AI_PROVIDER          # Should show: anthropic
echo $AI_MODEL             # Should show: claude-sonnet-4-20250514
echo ${ANTHROPIC_API_KEY:0:20}...  # Should show first 20 chars of your key

# Run Hyper
./hyper/coordinator  # or your usual start command
```

### Option 2: Direct Export

```bash
export AI_PROVIDER="anthropic"
export ANTHROPIC_API_KEY="sk-ant-api03-xxxxxxxxxxxxx"
export AI_MODEL="claude-sonnet-4-20250514"
export MAX_ITERATIONS=1000
export MAX_TOOL_CALLS=500
export MAX_OUT_TOKENS=8000
export TEMPERATURE=0.3

# Then run your application
./hyper/coordinator
```

---

## 🔍 Verification

After starting Hyper, check the logs for confirmation:

```bash
# You should see something like:
INFO  AI provider initialized  {"provider": "anthropic", "model": "claude-sonnet-4-20250514"}
```

---

## 🛠️ Advanced Configuration

### Temperature Control
Adjust model creativity/randomness (0.0 = deterministic, 1.0 = creative):
```bash
TEMPERATURE=0.3  # Current setting (recommended for code)
```

### Token Limits
```bash
MAX_OUT_TOKENS=8000      # Maximum output tokens per response
MAX_ITERATIONS=1000      # Maximum agentic reasoning iterations
MAX_TOOL_CALLS=500       # Maximum tool calls per session
```

### Fallback Model (Optional)
If you want to set a fallback model for rate limiting:
```bash
FALLBACK_MODEL="claude-3-sonnet-20240229"  # Older, more available model
```

---

## 🔄 Previous Configuration (Saved)

Your previous Ollama configuration has been commented out in `.env.hyper`:

```bash
# OpenAI/Ollama Configuration
# AI_PROVIDER="openai"
# OPENAI_BASE_URL="http://localhost:11434/v1"
# OPENAI_API_KEY="ollama"
# AI_MODEL="gpt-oss:120b-cloud"
```

To switch back, simply:
1. Comment out the Anthropic section
2. Uncomment the Ollama section
3. Restart Hyper

---

## ✅ Checklist

Before running Hyper with Anthropic:

- [ ] Obtain Anthropic API key from https://console.anthropic.com/
- [ ] Update `ANTHROPIC_API_KEY` in `.env.hyper` with your actual key
- [ ] Choose your preferred Claude model (current: `claude-sonnet-4-20250514`)
- [ ] Source the environment file: `source .env.hyper`
- [ ] Verify environment variables are set
- [ ] Start Hyper and check logs for successful initialization

---

## 📊 Code Configuration Details

The Anthropic provider is implemented in:
- **Provider code:** `hyper/internal/ai-service/provider.go:291-636`
- **Config loader:** `hyper/internal/ai-service/config.go:27-152`

### Supported Features ✅
- ✅ Streaming chat responses
- ✅ Tool calling (function calling)
- ✅ System prompts
- ✅ Multi-turn conversations
- ✅ Token limits and temperature control
- ✅ Claude 3, 3.5, and 4+ model families

### Environment Variables Used
```go
// Priority order for provider:
AI_PROVIDER  or  PROVIDER

// Priority order for API key:
ANTHROPIC_API_KEY  or  API_KEY

// Priority order for model:
AI_MODEL  or  MODEL  (defaults to "claude-3-sonnet-20240229")
```

---

## 🚨 Troubleshooting

### Error: "API_KEY or ANTHROPIC_API_KEY environment variable is required"
**Solution:** Make sure you've updated the API key in `.env.hyper` and sourced the file.

### Error: "Invalid API key"
**Solution:** Double-check your API key from the Anthropic console. Keys start with `sk-ant-api03-`.

### Error: "Rate limit exceeded"
**Solution:**
1. Wait a moment and try again
2. Set a `FALLBACK_MODEL` in your configuration
3. Upgrade your Anthropic plan for higher rate limits

### Hyper still using old model
**Solution:**
1. Make sure you sourced the file: `source .env.hyper`
2. Restart the Hyper service
3. Check environment variables: `env | grep -E "AI_|ANTHROPIC"`

---

## 💡 Tips

1. **Cost Management:** Claude Sonnet 4.5 is more expensive than earlier models. Monitor usage at https://console.anthropic.com/

2. **Context Window:** Claude supports 200K token context. Set appropriate limits:
   ```bash
   MAX_OUT_TOKENS=8000  # Adjust based on your needs
   ```

3. **Temperature:** For code generation, use lower values (0.1-0.3). For creative tasks, use higher values (0.7-1.0).

4. **Model Selection:** Start with `claude-sonnet-4-20250514` for best performance, fall back to `claude-3-5-sonnet-20241022` if needed.

---

## 📝 Next Steps

1. Add your Anthropic API key to `.env.hyper`
2. Source the file: `source .env.hyper`
3. Start Hyper
4. Test with a simple chat query
5. Monitor logs for any issues

Happy coding with Claude! 🎉
