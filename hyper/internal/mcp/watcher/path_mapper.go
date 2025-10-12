package watcher

import (
	"strings"

	"go.uber.org/zap"
)

// PathMapper handles translation between host and container paths
type PathMapper struct {
	mappings map[string]string // host -> container
	reverse  map[string]string // container -> host
	logger   *zap.Logger
}

// NewPathMapper creates a new path mapper from environment variable format
// Format: "/host/path1:/container/path1,/host/path2:/container/path2"
func NewPathMapper(mappingsEnv string, logger *zap.Logger) *PathMapper {
	mappings := make(map[string]string)
	reverse := make(map[string]string)

	if mappingsEnv == "" {
		logger.Info("No path mappings configured - paths will be used as-is")
		return &PathMapper{
			mappings: mappings,
			reverse:  reverse,
			logger:   logger,
		}
	}

	// Parse comma-separated mappings
	pairs := strings.Split(mappingsEnv, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			logger.Warn("Invalid path mapping format, skipping",
				zap.String("pair", pair))
			continue
		}

		host := strings.TrimSpace(parts[0])
		container := strings.TrimSpace(parts[1])

		if host == "" || container == "" {
			logger.Warn("Empty path in mapping, skipping",
				zap.String("pair", pair))
			continue
		}

		mappings[host] = container
		reverse[container] = host

		logger.Info("Registered path mapping",
			zap.String("host", host),
			zap.String("container", container))
	}

	logger.Info("Path mapper initialized",
		zap.Int("mappings", len(mappings)))

	return &PathMapper{
		mappings: mappings,
		reverse:  reverse,
		logger:   logger,
	}
}

// ToContainerPath translates a host path to container path
// If no mapping matches, returns the original path
func (pm *PathMapper) ToContainerPath(hostPath string) string {
	// Try each mapping (longest prefix match)
	bestMatch := ""
	bestContainer := hostPath

	for host, container := range pm.mappings {
		if strings.HasPrefix(hostPath, host) {
			// Use longest matching prefix
			if len(host) > len(bestMatch) {
				bestMatch = host
				bestContainer = strings.Replace(hostPath, host, container, 1)
			}
		}
	}

	if bestMatch != "" {
		pm.logger.Debug("Translated host path to container path",
			zap.String("host", hostPath),
			zap.String("container", bestContainer),
			zap.String("mapping", bestMatch))
	}

	return bestContainer
}

// ToHostPath translates a container path to host path
// If no mapping matches, returns the original path
func (pm *PathMapper) ToHostPath(containerPath string) string {
	// Try each reverse mapping (longest prefix match)
	bestMatch := ""
	bestHost := containerPath

	for container, host := range pm.reverse {
		if strings.HasPrefix(containerPath, container) {
			// Use longest matching prefix
			if len(container) > len(bestMatch) {
				bestMatch = container
				bestHost = strings.Replace(containerPath, container, host, 1)
			}
		}
	}

	if bestMatch != "" {
		pm.logger.Debug("Translated container path to host path",
			zap.String("container", containerPath),
			zap.String("host", bestHost),
			zap.String("mapping", bestMatch))
	}

	return bestHost
}

// HasMappings returns true if any path mappings are configured
func (pm *PathMapper) HasMappings() bool {
	return len(pm.mappings) > 0
}

// GetMappings returns all configured mappings (host -> container)
func (pm *PathMapper) GetMappings() map[string]string {
	// Return a copy to prevent modifications
	result := make(map[string]string, len(pm.mappings))
	for k, v := range pm.mappings {
		result[k] = v
	}
	return result
}

// ValidateContainerPath checks if a container path is accessible
// Returns true if path exists under any mapped container path or if no mappings configured
func (pm *PathMapper) ValidateContainerPath(containerPath string) bool {
	// If no mappings, all paths are valid (running on host)
	if !pm.HasMappings() {
		return true
	}

	// Check if path starts with any mapped container path
	for _, container := range pm.mappings {
		if strings.HasPrefix(containerPath, container) {
			return true
		}
	}

	return false
}
