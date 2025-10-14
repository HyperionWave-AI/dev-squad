package watcher

import (
    "reflect"
    "testing"

    "go.uber.org/zap"
)

func TestPathMapperBasic(t *testing.T) {
    logger := zap.NewNop()

    // No mappings
    pm := NewPathMapper("", logger)
    if pm.HasMappings() {
        t.Fatalf("expected no mappings")
    }
    if got := pm.ToContainerPath("/some/path"); got != "/some/path" {
        t.Fatalf("expected unchanged path, got %s", got)
    }
    if !pm.ValidateContainerPath("/any/path") {
        t.Fatalf("validate should be true when no mappings")
    }
    if m := pm.GetMappings(); len(m) != 0 {
        t.Fatalf("expected empty mappings, got %v", m)
    }
}

func TestPathMapperParsingAndTranslation(t *testing.T) {
    logger := zap.NewNop()
    env := "/host:/container,/host/long:/container/long"
    pm := NewPathMapper(env, logger)

    if !pm.HasMappings() {
        t.Fatalf("expected mappings to be present")
    }
    // Ensure both mappings are stored
    expected := map[string]string{"/host": "/container", "/host/long": "/container/long"}
    if !reflect.DeepEqual(pm.GetMappings(), expected) {
        t.Fatalf("mappings mismatch: got %v, want %v", pm.GetMappings(), expected)
    }

    // Longest prefix should be used for translation
    hostPath := "/host/long/sub/file.txt"
    container := pm.ToContainerPath(hostPath)
    if container != "/container/long/sub/file.txt" {
        t.Fatalf("longest prefix translation failed, got %s", container)
    }

    // Shorter prefix translation for non‑long path
    hostPath2 := "/host/other/file.txt"
    container2 := pm.ToContainerPath(hostPath2)
    if container2 != "/container/other/file.txt" {
        t.Fatalf("short prefix translation failed, got %s", container2)
    }

    // Reverse translation
    if host := pm.ToHostPath(container); host != hostPath {
        t.Fatalf("reverse translation failed, got %s", host)
    }
    if host := pm.ToHostPath(container2); host != hostPath2 {
        t.Fatalf("reverse translation failed for short path, got %s", host)
    }
}

func TestPathMapperInvalidEntries(t *testing.T) {
    logger := zap.NewNop()
    // Include an invalid entry and an empty mapping
    env := "badpair,/valid:/mapped,,/also/bad:"
    pm := NewPathMapper(env, logger)
    // Only the valid mapping should be kept
    if len(pm.GetMappings()) != 1 {
        t.Fatalf("expected only one valid mapping, got %d", len(pm.GetMappings()))
    }
    if _, ok := pm.GetMappings()["/valid"]; !ok {
        t.Fatalf("valid mapping missing")
    }
}

func TestValidateContainerPath(t *testing.T) {
    logger := zap.NewNop()
    env := "/host:/container"
    pm := NewPathMapper(env, logger)
    if !pm.ValidateContainerPath("/container/sub/file.go") {
        t.Fatalf("expected valid container path")
    }
    if pm.ValidateContainerPath("/other/path") {
        t.Fatalf("expected invalid container path")
    }
}
