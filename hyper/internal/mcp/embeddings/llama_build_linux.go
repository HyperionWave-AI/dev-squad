//go:build linux

package embeddings

// Linux build configuration for llama.cpp
// Supports CUDA, Vulkan, or CPU-only builds
//
// Build with CUDA (NVIDIA GPUs):
// export CGO_LDFLAGS="-L/usr/local/cuda/lib64 -lcublas -lcudart -lcuda"
// export CGO_CXXFLAGS="-DGGML_USE_CUBLAS -I/usr/local/cuda/include"
// go build
//
// Build with Vulkan (AMD/Intel/NVIDIA GPUs):
// export CGO_CXXFLAGS="-DGGML_USE_VULKAN"
// export CGO_LDFLAGS="-lvulkan"
// go build
//
// Build CPU-only (no GPU):
// go build
//
// Performance:
// - CUDA: ~2000-4000 embeddings/second (NVIDIA GPUs)
// - Vulkan: ~1000-2500 embeddings/second (all GPUs)
// - CPU: ~100-300 embeddings/second (depends on CPU)

import "C"
