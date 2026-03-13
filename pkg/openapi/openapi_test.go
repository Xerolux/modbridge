package openapi

import (
	"testing"
)

func TestGenerate(t *testing.T) {
	spec := Generate()
	if spec == nil {
		t.Fatal("Expected non-nil spec")
	}
	if spec.Info.Title != "ModBridge API" {
		t.Errorf("Expected title 'ModBridge API', got '%s'", spec.Info.Title)
	}
}

func TestSpec_ToJSON(t *testing.T) {
	spec := Generate()
	json, err := spec.ToJSON()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(json) == 0 {
		t.Error("Expected non-empty JSON")
	}
}
