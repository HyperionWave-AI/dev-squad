#!/bin/bash

# Download nomic-embed-text-v1.5 GGUF Model
# For embedded llama.cpp embeddings with GPU acceleration
# Model: nomic-embed-text-v1.5 (768 dimensions)
# Size: ~274MB (Q4_K_M quantized for optimal speed/quality)

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
MODEL_DIR="$PROJECT_ROOT/models"
MODEL_FILE="nomic-embed-text-v1.5.Q4_K_M.gguf"
MODEL_PATH="$MODEL_DIR/$MODEL_FILE"
MODEL_URL="https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Hyper - Embedding Model Download                         ║${NC}"
echo -e "${BLUE}║  nomic-embed-text-v1.5 (GGUF)                              ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Create models directory
mkdir -p "$MODEL_DIR"

# Check if model already exists
if [ -f "$MODEL_PATH" ]; then
    echo -e "${GREEN}✓ Model already downloaded${NC}"
    echo -e "  Location: $MODEL_PATH"
    echo -e "  Size: $(du -sh "$MODEL_PATH" | awk '{print $1}')"
    echo ""
    echo -e "${YELLOW}To re-download, delete the file first:${NC}"
    echo -e "  rm $MODEL_PATH"
    exit 0
fi

# Download model
echo -e "${YELLOW}Downloading nomic-embed-text-v1.5...${NC}"
echo -e "  Source: $MODEL_URL"
echo -e "  Target: $MODEL_PATH"
echo -e "  Size: ~274MB"
echo ""

curl -L --progress-bar "$MODEL_URL" -o "$MODEL_PATH"

if [ ! -f "$MODEL_PATH" ]; then
    echo -e "${RED}ERROR: Download failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Download completed successfully! ✓                       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Model location: ${YELLOW}$MODEL_PATH${NC}"
echo -e "Model size:     $(du -sh "$MODEL_PATH" | awk '{print $1}')"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo ""
echo -e "1. The model is now ready to use with Hyper"
echo -e "   ${YELLOW}./bin/hyper --mode=http${NC}"
echo ""
echo -e "2. Embeddings will be GPU-accelerated automatically:"
echo -e "   ${YELLOW}• macOS: Metal (M1/M2/M3)${NC}"
echo -e "   ${YELLOW}• Windows: CUDA (NVIDIA) or Vulkan (AMD/Intel)${NC}"
echo -e "   ${YELLOW}• Linux: CUDA or Vulkan${NC}"
echo ""
echo -e "${GREEN}Expected performance on your M3 Max:${NC}"
echo -e "  Throughput: ~2,000-4,000 embeddings/second"
echo -e "  Latency: 15-30ms per batch"
echo -e "  vs Voyage AI: 10-20x faster (no network latency!)"
echo ""
