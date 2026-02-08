package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

type Session struct {
	Config         *Config
	SessionDir     string
	CurrentPtrFile string
	LogFile        string
	Messages       []openai.ChatCompletionMessage
}

func NewSession(config *Config) *Session {
	sessionDir := config.Settings.SessionDir
	if sessionDir == "" {
		sessionDir = filepath.Join(config.SaaDir, "session")
	}
	return &Session{
		Config:         config,
		SessionDir:     sessionDir,
		CurrentPtrFile: filepath.Join(sessionDir, "current.json"),
	}
}

func (s *Session) initSessionDir() error {
	return os.MkdirAll(s.SessionDir, 0755)
}

func (s *Session) GetCurrentLogFile() (string, error) {
	data, err := os.ReadFile(s.CurrentPtrFile)
	if err != nil {
		return "", err
	}
	var ptr struct {
		LogFile string `json:"log_file"`
	}
	if err := json.Unmarshal(data, &ptr); err != nil {
		return "", err
	}
	return ptr.LogFile, nil
}

func (s *Session) Load() error {
	if err := s.initSessionDir(); err != nil {
		return err
	}

	data, err := os.ReadFile(s.CurrentPtrFile)
	if err == nil {
		var ptr struct {
			LogFile string `json:"log_file"`
		}
		if err := json.Unmarshal(data, &ptr); err == nil {
			s.LogFile = filepath.Join(s.SessionDir, ptr.LogFile)
			if _, err := os.Stat(s.LogFile); err == nil {
				return s.loadMessages()
			}
		}
	}

	return s.NewSession()
}

func (s *Session) NewSession() error {
	if err := s.initSessionDir(); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	u := uuid.New().String()[:8]
	filename := fmt.Sprintf("%s_%s.jsonl", timestamp, u)
	s.LogFile = filepath.Join(s.SessionDir, filename)

	ptrData, _ := json.Marshal(map[string]string{"log_file": filename})
	os.WriteFile(s.CurrentPtrFile, ptrData, 0644)

	systemPrompt, err := s.Config.ResolveSystemPrompt()
	if err != nil {
		return err
	}

	s.Messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}
	return s.Save()
}

func (s *Session) Clear() error {
	if err := os.RemoveAll(s.SessionDir); err != nil {
		return err
	}
	return s.initSessionDir()
}

func (s *Session) Switch(filename string) error {
	path := filepath.Join(s.SessionDir, filename)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("session file not found: %s", filename)
	}

	ptrData, err := json.Marshal(map[string]string{"log_file": filename})
	if err != nil {
		return err
	}
	return os.WriteFile(s.CurrentPtrFile, ptrData, 0644)
}

func (s *Session) List() ([]string, error) {
	files, err := os.ReadDir(s.SessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessionFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".jsonl" {
			sessionFiles = append(sessionFiles, f.Name())
		}
	}

	sort.Slice(sessionFiles, func(i, j int) bool {
		return sessionFiles[i] > sessionFiles[j]
	})

	return sessionFiles, nil
}

func (s *Session) loadMessages() error {
	file, err := os.Open(s.LogFile)
	if err != nil {
		return err
	}
	defer file.Close()

	s.Messages = []openai.ChatCompletionMessage{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg openai.ChatCompletionMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
			s.Messages = append(s.Messages, msg)
		}
	}
	return scanner.Err()
}

func (s *Session) Save() error {
	file, err := os.Create(s.LogFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, msg := range s.Messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if _, err := file.Write(data); err != nil {
			return err
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) AddMessage(msg openai.ChatCompletionMessage) error {
	s.Messages = append(s.Messages, msg)
	file, err := os.OpenFile(s.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if _, err := file.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

const SystemPrompt = `You are SAA (Single Action Agent).
You perform tasks autonomously by utilizing Bash commands to fulfill user instructions.

Available Tools:

1. bash(command): Executes a Bash command. Standard output, standard error, and exit code will be returned.

Constraints:

- You can only execute one tool at a time.
- Consider the next step after reviewing the results of the command.
- If a command fails, investigate the error, correct it, and retry.
- When the task is complete or if you need to ask the user a question, respond with a regular message without calling any tools.
`
