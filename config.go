package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const DefaultMaxOutput = 1000

type Config struct {
	ProjectRoot string   `json:"-"`
	SaaDir      string   `json:"-"`
	ConfigFile  string   `json:"-"`
	Settings    Settings `json:"settings"`
}

type Settings struct {
	APIKey           string `mapstructure:"api_key" json:"api_key,omitempty"`
	APIURL           string `mapstructure:"api_url" json:"api_url,omitempty"`
	Model            string `mapstructure:"model" json:"model,omitempty"`
	SessionDir       string `mapstructure:"session_dir" json:"session_dir,omitempty"`
	MaxStdout        int    `mapstructure:"max_stdout" json:"max_stdout,omitempty"`
	MaxStderr        int    `mapstructure:"max_stderr" json:"max_stderr,omitempty"`
	SystemPromptFile string `mapstructure:"system_prompt_file" json:"system_prompt_file,omitempty"`
	ShowToolCall     bool   `mapstructure:"show_tool_call" json:"show_tool_call,omitempty"`
	ShowToolResult   bool   `mapstructure:"show_tool_result" json:"show_tool_result,omitempty"`
	ShowReasoning    bool   `mapstructure:"show_reasoning" json:"show_reasoning,omitempty"`
	Verbose          bool   `mapstructure:"verbose" json:"verbose,omitempty"`
}

func (c *Config) ResolveSystemPrompt() (string, error) {
	file := c.Settings.SystemPromptFile
	if file == "" {
		return SystemPrompt, nil
	}

	// Relative to config.json
	if c.ConfigFile != "" {
		path := filepath.Join(filepath.Dir(c.ConfigFile), file)
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}

	// Relative to project root
	if c.ProjectRoot != "" {
		path := filepath.Join(c.ProjectRoot, file)
		if _, err := os.Stat(path); err == nil {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}

	// Absolute path or relative to CWD (as a fallback)
	content, err := os.ReadFile(file)
	if err == nil {
		return string(content), nil
	}

	return "", fmt.Errorf("system prompt file not found: %s", file)
}

func NewConfig() (*Config, error) {
	root, err := findProjectRoot()
	if err != nil {
		return nil, err
	}

	saaDir := filepath.Join(root, ".saa")
	configFile := filepath.Join(saaDir, "config.json")

	viper.SetEnvPrefix("SAA")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		home, _ := os.UserHomeDir()
		xdgConfigHome = filepath.Join(home, ".config")
	}
	userConfigPath := filepath.Join(xdgConfigHome, "saa", "config.json")

	loadFileToViper(userConfigPath)
	loadFileToViper(configFile)

	var settings Settings
	if err := viper.Unmarshal(&settings); err != nil {
		return nil, err
	}

	c := &Config{
		ProjectRoot: root,
		SaaDir:      saaDir,
		ConfigFile:  configFile,
		Settings:    settings,
	}

	return c, nil
}

func loadFileToViper(path string) {
	if _, err := os.Stat(path); err != nil {
		return
	}
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err == nil {
		viper.MergeConfigMap(v.AllSettings())
	}
}

var ErrProjectRootNotFound = fmt.Errorf("project root (.saa) not found")

func GetProjectRoot() (string, error) {
	if v := os.Getenv("SAA_PROJECT_ROOT"); v != "" {
		return filepath.Abs(v)
	}

	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(current, ".saa")); err == nil {
			return filepath.Abs(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", ErrProjectRootNotFound
}

func findProjectRoot() (string, error) {
	root, err := GetProjectRoot()
	if err == nil {
		return root, nil
	}
	return os.Getwd()
}

func (c *Config) EnsureSaaDir() error {
	return os.MkdirAll(c.SaaDir, 0755)
}

func (c *Config) SaveConfig() error {
	if err := c.EnsureSaaDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c.Settings, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.ConfigFile, data, 0644)
}

func (c *Config) Validate() error {
	var missing []string
	if c.Settings.APIURL == "" {
		missing = append(missing, "api_url")
	}
	if c.Settings.Model == "" {
		missing = append(missing, "model")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing configuration: %v. Use 'saa config' or environment variables (SAA_API_KEY, SAA_API_URL, SAA_MODEL) to set them", missing)
	}
	return nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", s)
	}
}
