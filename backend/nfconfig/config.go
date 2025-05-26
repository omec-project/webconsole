package nfconfig

type ServiceConfiguration struct {
	TLS struct {
		enabled bool   `yaml:"enabled"`
		Key     string `yaml:"keyPath"`
		Pem     string `yaml:"certPath"`
	} `yaml:"tls"`
}

// TODO: implement the config models in the next PRs

type AccessMobilityConfig struct{}

type PlmnConfig struct{}

type PlmnSnssaiConfig struct{}

type SessionManagementConfig struct{}

type PolicyControlConfig struct{}
