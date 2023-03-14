package config

type Config struct {
	Community    string     `yaml:"community"`
	Target       string     `yaml:"target"`
	Interface    *Interface `yaml:"interface"`
	Mibs         []string   `yaml:"mibs"`
	SkipLinkdown bool       `yaml:"skip-linkdown"`
	Name         string     `yaml:"name"`
	Mackerel     *Mackerel  `yaml:"mackerel"`
}

type Interface struct {
	Include *string `yaml:"include"`
	Exclude *string `yaml:"exclude"`
}

type Mackerel struct {
	HostID string `yaml:"host-id"`
	ApiKey string `yaml:"x-api-key"`
}
