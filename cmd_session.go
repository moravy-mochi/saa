package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			if _, err := os.Stat(session.SessionDir); os.IsNotExist(err) {
				return fmt.Errorf("session directory not found: %s", session.SessionDir)
			}
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all session history files",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			files, err := session.List()
			if err != nil {
				return err
			}

			for _, f := range files {
				fmt.Println(f)
			}
			return nil
		},
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "Show current session history file",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			logFile, err := session.GetCurrentLogFile()
			if err != nil {
				return nil
			}

			fmt.Println(logFile)
			return nil
		},
	}

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all session history files",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			if err := session.Clear(); err != nil {
				return err
			}

			return nil
		},
	}

	switchCmd := &cobra.Command{
		Use:   "switch [session-file]",
		Short: "Switch to a specific session history file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := NewConfig()
			if err != nil {
				return err
			}

			session := NewSession(config)
			if err := session.Switch(args[0]); err != nil {
				return err
			}

			fmt.Printf("Switched to session: %s\n", args[0])
			return nil
		},
	}

	stdoutCmd := &cobra.Command{
		Use:   "stdout [timestamp-uuid]",
		Short: "Show stdout log for a specific command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLog(args[0], "stdout")
		},
	}

	stderrCmd := &cobra.Command{
		Use:   "stderr [timestamp-uuid]",
		Short: "Show stderr log for a specific command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLog(args[0], "stderr")
		},
	}

	cmd.AddCommand(listCmd, currentCmd, clearCmd, switchCmd, stdoutCmd, stderrCmd)
	return cmd
}

func showLog(id, streamName string) error {
	config, err := NewConfig()
	if err != nil {
		return err
	}

	session := NewSession(config)

	filename := fmt.Sprintf("%s.%s.log", id, streamName)
	path := filepath.Join(session.SessionDir, filename)

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", path)
		}
		return err
	}

	fmt.Print(string(content))
	return nil
}
