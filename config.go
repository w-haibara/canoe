package canoe

import (
	"os"
	"strings"

	"github.com/kevinburke/ssh_config"
)

type Config struct {
	SSH SSHConfig
}

type SSHConfig struct {
	Host           string
	User           string
	Port           string
	PrivateKeyPath string
}

func LoadConfig(sshhost string) (Config, error) {
	sshcfg, err := LoadSSHConfig(sshhost)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		SSH: sshcfg,
	}

	return cfg, nil
}

func LoadSSHConfig(host string) (SSHConfig, error) {
	cfg := SSHConfig{Host: host}

	if cfg.User == "" {
		cfg.User = ssh_config.Get(cfg.Host, "User")
	}

	if cfg.Port == "" {
		cfg.Port = ssh_config.Get(cfg.Host, "Port")
	}

	if cfg.PrivateKeyPath == "" {
		path := ssh_config.Get(cfg.Host, "IdentityFile")
		if strings.HasPrefix(path, "~/") {
			d, err := os.UserHomeDir()
			if err != nil {
				return SSHConfig{}, err
			}

			path = strings.Replace(path, "~", d, 1)
		}
		cfg.PrivateKeyPath = path
	}

	return cfg, nil
}

func (cfg Config) SSHURL() string {
	return "ssh://" + cfg.SSH.User + "@" + cfg.SSH.Host + ":" + cfg.SSH.Port
}
