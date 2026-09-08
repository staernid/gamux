package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestGetConfigSchema(t *testing.T) {
	schema := GetConfigSchema()
	if len(schema) == 0 {
		t.Fatalf("expected non-empty schema, got 0 items")
	}

	keys := make(map[string]PropertyDescriptor)
	for _, prop := range schema {
		if prop.Key == "" {
			t.Errorf("found property descriptor with empty key: %+v", prop)
		}
		if prop.Label == "" {
			t.Errorf("property %q has empty label", prop.Key)
		}
		if prop.Description == "" {
			t.Errorf("property %q has empty description", prop.Key)
		}
		if prop.Category == "" {
			t.Errorf("property %q has empty category", prop.Key)
		}
		if prop.Type == "" {
			t.Errorf("property %q has empty type", prop.Key)
		}
		if prop.Scope != ScopeGlobal && prop.Scope != ScopeBoth {
			t.Errorf("property %q expected scope 'global' or 'both', got %q", prop.Key, prop.Scope)
		}

		if prop.Type == "select" && len(prop.Options) == 0 {
			t.Errorf("select property %q must have at least one option", prop.Key)
		}

		if _, exists := keys[prop.Key]; exists {
			t.Errorf("duplicate property key %q in schema", prop.Key)
		}
		keys[prop.Key] = prop
	}

	// Verify top-level fields of Config are mapped
	cfgType := reflect.TypeOf(Config{})
	for i := 0; i < cfgType.NumField(); i++ {
		field := cfgType.Field(i)
		jsonTag := field.Tag.Get("json")
		parts := strings.Split(jsonTag, ",")
		tagKey := parts[0]
		if tagKey == "" || tagKey == "-" {
			continue
		}

		if tagKey == "multiseat" {
			// Check nested Multiseat fields
			msType := reflect.TypeOf(MultiseatConfig{})
			for j := 0; j < msType.NumField(); j++ {
				msField := msType.Field(j)
				msParts := strings.Split(msField.Tag.Get("json"), ",")
				msTagKey := msParts[0]
				if msTagKey == "" || msTagKey == "-" {
					continue
				}
				expectedKey := "multiseat." + msTagKey
				if _, ok := keys[expectedKey]; !ok {
					t.Errorf("multiseat field %q missing from schema", expectedKey)
				}
			}
			continue
		}

		if _, ok := keys[tagKey]; !ok {
			t.Errorf("config field %q missing from schema", tagKey)
		}
	}
}

func TestGetGameConfigSchema(t *testing.T) {
	schema := GetGameConfigSchema()
	if len(schema) == 0 {
		t.Fatalf("expected non-empty game config schema, got 0 items")
	}

	keys := make(map[string]PropertyDescriptor)
	for _, prop := range schema {
		if prop.Key == "" {
			t.Errorf("found property descriptor with empty key: %+v", prop)
		}
		if prop.Label == "" {
			t.Errorf("property %q has empty label", prop.Key)
		}
		if prop.Description == "" {
			t.Errorf("property %q has empty description", prop.Key)
		}
		if prop.Category == "" {
			t.Errorf("property %q has empty category", prop.Key)
		}
		if prop.Type == "" {
			t.Errorf("property %q has empty type", prop.Key)
		}
		if prop.Scope != ScopeGame && prop.Scope != ScopeBoth {
			t.Errorf("property %q expected scope 'game' or 'both', got %q", prop.Key, prop.Scope)
		}

		if prop.Type == "select" && len(prop.Options) == 0 {
			t.Errorf("select property %q must have at least one option", prop.Key)
		}

		if _, exists := keys[prop.Key]; exists {
			t.Errorf("duplicate property key %q in game schema", prop.Key)
		}
		keys[prop.Key] = prop
	}

	expectedKeys := []string{"runner", "exe_args", "language", "mangohud", "gamescope", "gamescope_fps", "disable_overlay", "offline_mode", "wine_prefix", "dll_overrides", "custom_env"}
	for _, k := range expectedKeys {
		if _, ok := keys[k]; !ok {
			t.Errorf("expected key %q missing from game schema", k)
		}
	}

	// Verify inheritance linkage
	if keys["runner"].InheritsFrom != "default_runner" {
		t.Errorf("expected runner to inherit from 'default_runner', got %q", keys["runner"].InheritsFrom)
	}
	if keys["mangohud"].InheritsFrom != "mangohud" {
		t.Errorf("expected mangohud to inherit from 'mangohud', got %q", keys["mangohud"].InheritsFrom)
	}
}
