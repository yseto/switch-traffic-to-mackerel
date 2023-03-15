package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

func (c *Config) Save(hostID string) error {
	stat, err := os.Stat(loadedFilename)
	if err != nil {
		return err
	}
	perm := stat.Mode()

	// read original config
	f, err := os.ReadFile(loadedFilename)
	if err != nil {
		return err
	}
	var t YAMLConfig
	err = yaml.Unmarshal(f, &t)
	if err != nil {
		return err
	}

	// added Mackerel.HostID
	t.Mackerel.HostID = hostID

	b, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(loadedFilename, b, perm)
}
