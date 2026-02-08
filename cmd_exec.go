package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	maxStdout        int
	maxStderr        int
	systemPromptFile string
)

func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec [prompt]",
		Aliases: []string{"x"},
		Short:   "Execute a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("show-tool-call") {
				config.Settings.ShowToolCall = showToolCall
			}
			if cmd.Flags().Changed("show-tool-result") {
				config.Settings.ShowToolResult = showToolResult
			}
			if cmd.Flags().Changed("show-reasoning") {
				config.Settings.ShowReasoning = showReasoning
			}
			if cmd.Flags().Changed("verbose") {
				config.Settings.Verbose = verbose
			}

			if err := config.Validate(); err != nil {
				return err
			}

			session := NewSession(config)
			if err := session.Load(); err != nil {
				return err
			}

			var parts []string
			fi, _ := os.Stdin.Stat()
			if (fi.Mode() & os.ModeCharDevice) == 0 {
				bytes, _ := io.ReadAll(os.Stdin)
				if content := strings.TrimSpace(string(bytes)); content != "" {
					parts = append(parts, content)
				}
			}
			if len(args) > 0 {
				parts = append(parts, strings.Join(args, " "))
			}

			prompt := strings.Join(parts, " ")
			if prompt == "" {
				return fmt.Errorf("no prompt provided")
			}

			agent := NewAgent(config, session)
			return agent.Run(prompt)
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().IntVar(&maxStdout, "max-stdout", DefaultMaxOutput, "Maximum characters for stdout before truncation. Use -1 for no limit.")
	cmd.Flags().IntVar(&maxStderr, "max-stderr", DefaultMaxOutput, "Maximum characters for stderr before truncation. Use -1 for no limit.")
	cmd.Flags().StringVar(&systemPromptFile, "system-prompt", "", "File containing the system prompt")

	viper.BindPFlag("max_stdout", cmd.Flags().Lookup("max-stdout"))
	viper.BindPFlag("max_stderr", cmd.Flags().Lookup("max-stderr"))
	viper.BindPFlag("system_prompt_file", cmd.Flags().Lookup("system-prompt"))

	return cmd
}
