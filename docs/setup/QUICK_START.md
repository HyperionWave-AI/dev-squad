# Quick Start - Local Ollama + Qwen Coder

This is the fastest local setup for running Hyperion with:
- Chat model: `qwen2.5-coder` via Ollama
- Embeddings: `nomic-embed-text` via Ollama
- Local storage: MongoDB + Qdrant (Docker)

## Prerequisites

- Docker + Docker Compose
- `hyper` binary available (`hyper` in `PATH` or full path to `bin/hyper`)

## 1) Initialize a local project

```bash
mkdir -p ~/hyperion-local
cd ~/hyperion-local
hyper init -provider ollama
```

This creates:
- `docker-compose.yml`
- `.env.hyper`
- `litellm.config.yaml`
- `HYPER_README.md`

## 2) Start local services

```bash
docker compose up -d
docker compose logs -f ollama-pull
```

Wait until `ollama-pull` finishes and confirms the embedding model download.

## 3) Pull Qwen Coder in Ollama

`hyper init` pulls the embedding model by default, but not the chat model.

```bash
docker exec hyper-ollama ollama pull qwen2.5-coder:7b
docker exec hyper-ollama ollama list
```

You can also use a larger model (example: `qwen2.5-coder:14b`) if your machine has enough RAM/VRAM.

## 4) Configure `.env.hyper` for local Ollama chat

Ensure these values are set in `.env.hyper`:

```bash
# Chat / agent model through Ollama's OpenAI-compatible endpoint
AI_PROVIDER=openai
OPENAI_BASE_URL=http://localhost:7335/v1
OPENAI_API_KEY=ollama
AI_MODEL=qwen2.5-coder:7b

# Embeddings
EMBEDDING=ollama
OLLAMA_URL=http://localhost:7335
OLLAMA_MODEL=nomic-embed-text
```

## 5) Run Hyperion

```bash
hyper --mode=http --config=.env.hyper
```

## 6) Verify

```bash
# Hyper health
curl http://localhost:7095/api/v1/health

# Ollama models
curl http://localhost:7335/api/tags
```

Open UI at: **http://localhost:7095**

## Optional: Native Ollama (no Docker Ollama)

If you run Ollama directly on your machine (`http://localhost:11434`), use:

```bash
OPENAI_BASE_URL=http://localhost:11434/v1
OLLAMA_URL=http://localhost:11434
```

## Troubleshooting

- `model "... not found"`: run `docker exec hyper-ollama ollama pull qwen2.5-coder:7b`
- `failed to connect to Ollama`: check `docker logs hyper-ollama`
- Slow responses: use a smaller model (`qwen2.5-coder:7b`) and reduce concurrency

## More Information

- `docs/setup/HYPER_INIT_GUIDE.md`
- `docs/setup/HYPER_INIT_WITH_PROVIDER.md`
- `docs/guides/README-HYPER.md`
