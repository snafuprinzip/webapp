package webapp

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type ConfigStruct struct {
	BindAddress      string   `yaml:"bindAddress"`
	DBConnector      string   `yaml:"dbConnector"`
	DataDirectory    string   `yaml:"dataDirectory"`
	LogDirectory     string   `yaml:"logDirectory"`
	LogLevel         Loglevel `yaml:"logLevel"`
	OpenRegistration bool     `yaml:"openRegistration"`
}

var (
	Config *ConfigStruct
)

func ReadConfig(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		Logf(ErrorLevel, "%s\n", err)
		Logf(ErrorLevel, "Unable to open configuration file for reading.\nUsing default configuration\n")
		Config = &ConfigStruct{BindAddress: ":3000", DataDirectory: "./data/", LogDirectory: "/var/log/", OpenRegistration: true}
		Config.Save(filename)
		return err
	}

	err = yaml.Unmarshal(file, &Config)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func (c *ConfigStruct) Yaml() []byte {
	y, err := yaml.Marshal(c)
	if err != nil {
		Logf(ErrorLevel, "%s\n", err)
		return nil
	}
	return y
}

func (c *ConfigStruct) Save(path string) {
	err := os.WriteFile(path, c.Yaml(), 0600)
	if err != nil {
		Logf(ErrorLevel, "Unable to save configuration to file %s.\n%s\n", path, err)
	}
}
