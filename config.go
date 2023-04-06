package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-yaml/yaml"
)

var (
	Config *ConfigStruct
)

type ConfigStruct struct {
	BindAddress      string `yaml:"bindAddress"`
	DBConnector      string `yaml:"dbConnector"`
	DataDirectory    string `yaml:"dataDirectory"`
	OpenRegistration bool   `yaml:"openRegistration"`
}

func ReadConfig(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("%s\n", err)
		log.Printf("Unable to open configuration file for reading.\nUsing default configuration\n")
		Config = &ConfigStruct{BindAddress: ":3000", DataDirectory: "./data/"}
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
		log.Printf("%s\n", err)
		return nil
	}
	return y
}

func (c *ConfigStruct) Save(path string) {
	err := os.WriteFile(path, c.Yaml(), 0600)
	if err != nil {
		log.Printf("Unable to save configuration to file %s.\n%s\n", path, err)
	}
}
