package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	apiKeyOverride     string
	apiURLOverride     string
	modelOverride      string
	sessionDirOverride string
	showToolCall       bool
	showToolResult     bool
	showReasoning      bool
	verbose            bool
)

type toggleBool struct {
	p *bool
}

func (b *toggleBool) Set(s string) error {
	v, err := parseBool(s)
	if err != nil {
		return err
	}
	*b.p = v
	return nil
}

func (b *toggleBool) String() string {
	return fmt.Sprintf("%v", *b.p)
}

func (b *toggleBool) Type() string {
	return "bool"
}

func (b *toggleBool) IsBoolFlag() bool {
	return true
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		os.Exit(0)
	}()

	rootCmd := &cobra.Command{
		Use:           "saa",
		Short:         "SAA: Single Action Agent",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.PersistentFlags().StringVar(&apiKeyOverride, "api-key", "", "OpenAI API Key")
	rootCmd.PersistentFlags().StringVar(&apiURLOverride, "api-url", "", "OpenAI Base URL")
	rootCmd.PersistentFlags().StringVar(&modelOverride, "model", "", "Model name")
	rootCmd.PersistentFlags().StringVar(&sessionDirOverride, "session-dir", "", "Directly specify the session directory")
	rootCmd.PersistentFlags().Var(&toggleBool{&showToolCall}, "show-tool-call", "Show agent's tool calls (bash commands)")
	rootCmd.PersistentFlags().Var(&toggleBool{&showToolResult}, "show-tool-result", "Show results of tool calls")
	rootCmd.PersistentFlags().Var(&toggleBool{&showReasoning}, "show-reasoning", "Show agent's reasoning content")
	rootCmd.PersistentFlags().VarP(&toggleBool{&verbose}, "verbose", "v", "Show all (tool calls, results, reasoning)")

	rootCmd.PersistentFlags().Lookup("show-tool-call").NoOptDefVal = "true"
	rootCmd.PersistentFlags().Lookup("show-tool-result").NoOptDefVal = "true"
	rootCmd.PersistentFlags().Lookup("show-reasoning").NoOptDefVal = "true"
	rootCmd.PersistentFlags().Lookup("verbose").NoOptDefVal = "true"

	viper.BindPFlag("api_key", rootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
	viper.BindPFlag("session_dir", rootCmd.PersistentFlags().Lookup("session-dir"))

	initCmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize .saa directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir, _ := os.Getwd()
			if len(args) > 0 {
				var err error
				targetDir, err = filepath.Abs(args[0])
				if err != nil {
					return err
				}
			}

			saaDir := filepath.Join(targetDir, ".saa")
			if _, err := os.Stat(saaDir); err == nil {
				return fmt.Errorf(".saa directory already exists at %s", saaDir)
			}

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			config, err := NewConfig()
			if err != nil {
				return err
			}
			config.SaaDir = saaDir
			if err := config.EnsureSaaDir(); err != nil {
				return err
			}

			session := NewSession(config)
			if err := session.NewSession(); err != nil {
				return err
			}

			return nil
		},
	}

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configure SAA settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			if cmd.Flags().NFlag() == 0 && len(args) == 0 {
				data, err := json.MarshalIndent(config.Settings, "", "    ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			if _, err := os.Stat(config.SaaDir); os.IsNotExist(err) {
				return fmt.Errorf(".saa directory not found. Run 'saa init' first")
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

			if err := config.SaveConfig(); err != nil {
				return err
			}

			return nil
		},
	}

	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Start a new session",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			if err := session.NewSession(); err != nil {
				return err
			}

			return nil
		},
	}

	whereCmd := &cobra.Command{
		Use:   "where",
		Short: "Show the absolute path of the project root",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := GetProjectRoot()
			if err != nil {
				if errors.Is(err, ErrProjectRootNotFound) {
					return nil
				}
				return err
			}
			fmt.Println(root)
			return nil
		},
	}

	rootCmd.AddCommand(initCmd, configCmd, NewExecCmd(), newCmd, NewSessionCmd(), whereCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
