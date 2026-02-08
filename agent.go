package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

type Agent struct {
	Config  *Config
	Session *Session
	Client  *openai.Client
}

func NewAgent(config *Config, session *Session) *Agent {
	clientConfig := openai.DefaultConfig(config.Settings.APIKey)
	if config.Settings.APIURL != "" {
		clientConfig.BaseURL = config.Settings.APIURL
	}
	client := openai.NewClientWithConfig(clientConfig)

	return &Agent{
		Config:  config,
		Session: session,
		Client:  client,
	}
}

func (a *Agent) Run(prompt string) error {
	verbose := a.Config.Settings.Verbose
	showCall := verbose || a.Config.Settings.ShowToolCall
	showResult := verbose || a.Config.Settings.ShowToolResult
	showReasoning := verbose || a.Config.Settings.ShowReasoning
	anyShow := showCall || showResult || showReasoning

	if err := a.Session.AddMessage(openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: prompt,
	}); err != nil {
		return err
	}

	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "bash",
				Description: "Execute a bash command and get the output.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"command": {"type": "string", "description": "The command to run."},
						"timeout": {"type": "integer", "description": "The timeout in seconds. Default is no timeout."}
					},
					"required": ["command"]
				}`),
			},
		},
	}

	for {
		resp, err := a.Client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    a.Config.Settings.Model,
				Messages: a.Session.Messages,
				Tools:    tools,
			},
		)
		if err != nil {
			return err
		}

		msg := resp.Choices[0].Message
		if err := a.Session.AddMessage(msg); err != nil {
			return err
		}

		if showReasoning && msg.ReasoningContent != "" {
			fmt.Printf("[REASONING]\n%s\n", msg.ReasoningContent)
		}

		if msg.Content != "" {
			if anyShow {
				fmt.Print("[MESSAGE]\n")
			}
			out := msg.Content
			if !strings.HasSuffix(out, "\n") {
				out += "\n"
			}
			fmt.Print(out)
		}

		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				if tc.Function.Name == "bash" {
					var args struct {
						Command string `json:"command"`
						Timeout int    `json:"timeout"`
					}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
						return err
					}

					if showCall {
						fmt.Printf("[TOOL] %s\n", args.Command)
					}

					timeout := time.Duration(args.Timeout) * time.Second
					res, err := ExecuteBash(args.Command, a.Config.ProjectRoot, timeout)
					var result string
					if err != nil {
						result = fmt.Sprintf("Error executing bash: %v", err)
					} else {
						stdout, err := a.handleOutput(res.Stdout, a.Config.Settings.MaxStdout, "stdout")
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error handling stdout: %v\n", err)
							stdout = res.Stdout
						}

						stderr, err := a.handleOutput(res.Stderr, a.Config.Settings.MaxStderr, "stderr")
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error handling stderr: %v\n", err)
							stderr = res.Stderr
						}

						result = fmt.Sprintf("Exit Code: %d\nSTDOUT:\n%s\nSTDERR:\n%s",
							res.ExitCode, stdout, stderr)
					}

					if showResult {
						fmt.Printf("[RESULT]\n%s\n", result)
					}

					if err := a.Session.AddMessage(openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    result,
						ToolCallID: tc.ID,
					}); err != nil {
						return err
					}
				}
			}
		} else {
			break
		}
	}

	return nil
}

func (a *Agent) handleOutput(content string, limit int, streamName string) (string, error) {
	if limit == 0 {
		limit = DefaultMaxOutput
	}
	if limit == -1 || len(content) <= limit {
		return content, nil
	}

	timestamp := time.Now().Format("20060102-150405")
	u := uuid.New().String()[:8]
	id := fmt.Sprintf("%s-%s", timestamp, u)
	filename := fmt.Sprintf("%s.%s.log", id, streamName)
	path := filepath.Join(a.Session.SessionDir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}

	truncated := content[:limit]
	msg := fmt.Sprintf("\n... (truncated. Use `saa session %s %s` to view full log.)", streamName, id)
	return truncated + msg, nil
}
