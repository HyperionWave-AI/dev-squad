//go:build windows

package embeddings

// Windows build configuration for llama.cpp
// Supports multiple GPU backends: CUDA (NVIDIA), Vulkan (AMD/Intel/NVIDIA)
//
// Build with CUDA (NVIDIA GPUs):
// set CGO_LDFLAGS=-L"C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\lib\x64" -lcublas -lcudart
// set CGO_CXXFLAGS=-DGGML_USE_CUBLAS -I"C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\include"
// go build
//
// Build with Vulkan (AMD/Intel/NVIDIA GPUs - broader compatibility):
// Download Vulkan SDK from https://vulkan.lunarg.com/sdk/home#windows
// set CGO_CXXFLAGS=-DGGML_USE_VULKAN -I"C:\VulkanSDK\1.3.xxx\Include"
// set CGO_LDFLAGS=-L"C:\VulkanSDK\1.3.xxx\Lib" -lvulkan-1
// go build
//
// Performance:
// - CUDA: ~1500-3000 embeddings/second (NVIDIA GPUs)
// - Vulkan: ~1000-2000 embeddings/second (all GPUs)
//
// Note: If no GPU found, falls back to CPU automatically

import "C"
