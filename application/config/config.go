package config

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed app_config.yaml
var configFile embed.FS

// Config holds the application configuration
type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
	} `yaml:"database"`
}

// Global variable to hold the config
var AppConfig Config

// LoadConfig reads the configuration file and parses it
func LoadConfig() error {
	data, err := configFile.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	err = decoder.Decode(&AppConfig)
	if err != nil {
		return fmt.Errorf("error decoding config file: %v", err)
	}

	return nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return &AppConfig
}
