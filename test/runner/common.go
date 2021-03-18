package runner

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// ConfigFile is the config file
type ConfigFile struct {
	Server Server `yaml:"server"`
	Data   `yaml:",inline"`
}

// Server is the connection details
type Server struct {
	URL  string `yaml:"url"`
	Port string `yaml:"port"`
}

// Data of BMC resources
type Data struct {
	Resources []Resource `yaml:"resources"`
}

// UseCases classes with names
type UseCases struct {
	Power  []string `yaml:"power"`
	Device []string `yaml:"device"`
}

// Resource details of a single BMC
type Resource struct {
	IP       string   `yaml:"ip"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Vendor   string   `yaml:"vendor"`
	UseCases UseCases `yaml:"useCases"`
}

// Config for the resources file
func (c *ConfigFile) Config(name string) {
	err := c.parseConfig(name)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// parseConfig reads and validates a config
func (c *ConfigFile) parseConfig(name string) error {
	config, err := ioutil.ReadFile(name)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(config, &c)
	if err != nil {
		return err
	}

	return c.validateConfig()
}

func (c *ConfigFile) validateConfig() error {
	return nil
}
