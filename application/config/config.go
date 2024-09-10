package config

import (
	"fmt"

	"github.com/spf13/viper"
)

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
	viper.SetConfigFile("application/config/config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		return fmt.Errorf("error unmarshaling config: %v", err)
	}

	return nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return &AppConfig
}
