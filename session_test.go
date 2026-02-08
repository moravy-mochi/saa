package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestSessionLifecycle(t *testing.T) {
	// Setup temp environment
	tmpDir, err := os.MkdirTemp("", "saa-test-session")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	saaDir := filepath.Join(tmpDir, ".saa")
	os.Mkdir(saaDir, 0755)

	config := &Config{
		SaaDir: saaDir,
		Settings: Settings{
			SessionDir: "", // use default
		},
	}

	// 1. New Session
	session := NewSession(config)
	if err := session.NewSession(); err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}

	if _, err := os.Stat(session.SessionDir); os.IsNotExist(err) {
		t.Errorf("session dir not created")
	}

	logFile, err := session.GetCurrentLogFile()
	if err != nil {
		t.Fatalf("GetCurrentLogFile failed: %v", err)
	}
	if logFile == "" {
		t.Error("log file name is empty")
	}

	// 2. Add Message
	msg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "Hello",
	}
	if err := session.AddMessage(msg); err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	// Verify file content
	fullPath := filepath.Join(session.SessionDir, logFile)

	// Let's decode properly
	messages, err := readMessages(fullPath)
	if err != nil {
		t.Fatalf("failed to read messages: %v", err)
	}

	// Expect at least 2 messages: System prompt (added in NewSession) + User message
	if len(messages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(messages))
	}
	lastMsg := messages[len(messages)-1]
	if lastMsg.Content != "Hello" {
		t.Errorf("expected last message 'Hello', got '%s'", lastMsg.Content)
	}

	// 3. List
	files, err := session.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 session file, got %d", len(files))
	}
	if files[0] != logFile {
		t.Errorf("expected file %s, got %s", logFile, files[0])
	}

	// 4. New Session (Rotate)
	if err := session.NewSession(); err != nil {
		t.Fatalf("NewSession (2) failed: %v", err)
	}
	newLogFile, _ := session.GetCurrentLogFile()
	if newLogFile == logFile {
		t.Error("session file did not change after NewSession")
	}

	// 5. Switch
	if err := session.Switch(logFile); err != nil {
		t.Fatalf("Switch failed: %v", err)
	}
	current, _ := session.GetCurrentLogFile()
	if current != logFile {
		t.Errorf("expected current %s, got %s", logFile, current)
	}

	// 6. Clear
	if err := session.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	files, _ = session.List()
	if len(files) != 0 {
		t.Errorf("expected 0 files after clear, got %d", len(files))
	}
}

func readMessages(path string) ([]openai.ChatCompletionMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var msgs []openai.ChatCompletionMessage
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var msg openai.ChatCompletionMessage
		if err := decoder.Decode(&msg); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}
