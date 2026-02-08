package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestGetProjectRoot(t *testing.T) {
	// Create a temporary directory structure
	// /tmp/saa-test/
	//   .saa/
	//   subdir/
	tmpDir, err := os.MkdirTemp("", "saa-test-root")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	saaDir := filepath.Join(tmpDir, ".saa")
	if err := os.Mkdir(saaDir, 0755); err != nil {
		t.Fatalf("failed to create .saa dir: %v", err)
	}

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Test from root
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	root, err := GetProjectRoot()
	if err != nil {
		t.Errorf("GetProjectRoot failed from root: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}

	// Test from subdir
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	root, err = GetProjectRoot()
	if err != nil {
		t.Errorf("GetProjectRoot failed from subdir: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}

	// Test from outside (should fail or return error)
	outsideDir, err := os.MkdirTemp("", "saa-outside")
	if err != nil {
		t.Fatalf("failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)
	if err := os.Chdir(outsideDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	// We expect ErrProjectRootNotFound, but since findProjectRoot falls back to Getwd(),
	// we need to check how GetProjectRoot behaves directly.
	_, err = GetProjectRoot()
	if err != ErrProjectRootNotFound {
		t.Errorf("expected ErrProjectRootNotFound, got %v", err)
	}
}

func TestConfigLoading(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "saa-test-config")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	saaDir := filepath.Join(tmpDir, ".saa")
	if err := os.Mkdir(saaDir, 0755); err != nil {
		t.Fatalf("failed to create .saa dir: %v", err)
	}

	configFile := filepath.Join(saaDir, "config.json")
	configContent := `{
		"api_key": "test_key",
		"model": "gpt-4",
		"max_stdout": 500
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Reset viper
	viper.Reset()

	config, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	if config.Settings.APIKey != "test_key" {
		t.Errorf("expected APIKey 'test_key', got '%s'", config.Settings.APIKey)
	}
	if config.Settings.Model != "gpt-4" {
		t.Errorf("expected Model 'gpt-4', got '%s'", config.Settings.Model)
	}
	if config.Settings.MaxStdout != 500 {
		t.Errorf("expected MaxStdout 500, got %d", config.Settings.MaxStdout)
	}

	// Test Environment Variable Override
	os.Setenv("SAA_MODEL", "gpt-3.5-turbo")
	defer os.Unsetenv("SAA_MODEL")

	// Need to reload config to pick up env vars (NewConfig does this)
	viper.Reset()
	config, err = NewConfig()
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	if config.Settings.Model != "gpt-3.5-turbo" {
		t.Errorf("expected Model 'gpt-3.5-turbo' from env, got '%s'", config.Settings.Model)
	}
}

func TestResolveSystemPrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "saa-test-prompt")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	saaDir := filepath.Join(tmpDir, ".saa")
	if err := os.Mkdir(saaDir, 0755); err != nil {
		t.Fatalf("failed to create .saa dir: %v", err)
	}

	// Create a custom prompt file
	promptFile := filepath.Join(saaDir, "custom_prompt.txt")
	expectedContent := "You are a test agent."
	if err := os.WriteFile(promptFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Create config pointing to it (relative path)
	configFile := filepath.Join(saaDir, "config.json")
	configContent := `{
		"system_prompt_file": "custom_prompt.txt"
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	viper.Reset()
	config, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	content, err := config.ResolveSystemPrompt()
	if err != nil {
		t.Fatalf("ResolveSystemPrompt failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("expected prompt content '%s', got '%s'", expectedContent, content)
	}
}
