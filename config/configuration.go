package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

const (
	ALL        uint8 = iota // All repos will be considered
	CONFIG                  // Only repos specified in the config will be considered
	ALLOW_LIST              // Only repos with the .pipelines file will be considered
	DENY_LIST               // Repos with the .nopipelines file will be ignored
)

type Config struct {
	Version string `yaml:"version"`
	// which repos should be considered
	Coverate uint8 `yaml:"coverage"`
	// Prefix used in the PR title
	PRPrefix string `yaml:"pr_prefix"`
	// Prefix used in the Branch
	BranchPrefix string `yaml:"branch_prefix"`
}

func New() Config {
	return Config{
		Version:      "0.1",
		Coverate:     ALL,
		PRPrefix:     "gitops-toolbox",
		BranchPrefix: "gitops-toolbox/",
	}
}

func Load() (Config, error) {
	data, err := os.ReadFile("./.gt-cicd.yaml")
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func Init() error {
	config := New()
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile("./.gt-cicd.yaml", data, 0644)
}
