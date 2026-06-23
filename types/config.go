package types

import (
	"github.com/agenthands/rod-cli/utils"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

const ConfigName = "rod-cli.yaml"

type Config struct {
	Mode           Mode         `yaml:"mode" json:"mode"`
	CDPEndpoint    string       `yaml:"cdpEndpoint" json:"cdpEndpoint"`
	BrowserBinPath string       `yaml:"browserBinPath" json:"browserBinPath"`
	Headless       bool         `yaml:"headless" json:"headless"`
	BrowserTempDir string       `yaml:"browserTempDir" json:"browserTempDir"`
	NoSandbox      bool         `yaml:"noSandbox" json:"noSandbox"`
	Proxy          string       `yaml:"proxy" json:"proxy"`
	LoggerConfig   LoggerConfig `yaml:"loggerConfig" json:"loggerConfig"`
	Raw            bool         `yaml:"raw" json:"raw"`
	Json           bool         `yaml:"json" json:"json"`
}

var (
	DefaultBrowserTempDir = "./rod/browser"

	DefaultConfig = Config{
		BrowserBinPath: "",
		Headless:       false,
		BrowserTempDir: DefaultBrowserTempDir,
		NoSandbox:      false,
		Proxy:          "",
		LoggerConfig:   DefaultLoggerConfig,
		Mode:           Text,
		Raw:            false,
		Json:           false,
	}
)

// InitDefaultConfig Generate the default configuration file
func InitDefaultConfig() error {

	// First, check if the configuration file exists at the default path. If it exists, do not generate the default configuration file.
	defaultConfigPath := filepath.Join("./", ConfigName)
	if exist, _ := utils.PathExists(defaultConfigPath); exist {
		return nil
	}

	// if default config file not exist, create it
	defaultConfig, err := os.Create(defaultConfigPath)
	if err != nil {
		return err
	}

	encoder := yaml.NewEncoder(defaultConfig)
	defer encoder.Close()

	err = encoder.Encode(DefaultConfig)
	if err != nil {
		return err
	}
	return nil
}

// LoadConfig Actually load the configuration file
// if ConfigPath is empty, generate the default configuration file in the current directory
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = filepath.Join("./", ConfigName)
		if err := InitDefaultConfig(); err != nil {
			return nil, errors.Wrapf(err, "init default config failed")
		}
	}

	// check if config file exist
	exist, err := utils.PathExists(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not open config file")
	}

	if exist {
		// validate config file name
		fileName := utils.FileName(configPath)
		if strings.Contains(ConfigName, fileName) {
			file, err := os.Open(configPath)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			decoder := yaml.NewDecoder(file)
			var config Config
			if err := decoder.Decode(&config); err != nil {
				return nil, err
			}
			return &config, nil
		}
		return nil, errors.Wrapf(err, "config file name is wrong")
	}
	return nil, errors.Wrapf(err, "config path %s not found", configPath)
}
