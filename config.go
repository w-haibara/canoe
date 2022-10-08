package canoe

import (
	"fmt"
	"os"
	"strings"

	"github.com/kevinburke/ssh_config"
)

type Config struct {
	SSHHost           string
	SSHUser           string
	SSHPort           string
	SSHPrivateKeyPath string
}

func LoadConfig(cfg Config) (Config, error) {
	if cfg.SSHHost == "" {
		return Config{}, fmt.Errorf("ssh host is needed")
	}

	if cfg.SSHUser == "" {
		cfg.SSHUser = ssh_config.Get(cfg.SSHHost, "User")
	}

	if cfg.SSHPort == "" {
		cfg.SSHPort = ssh_config.Get(cfg.SSHHost, "Port")
	}

	if cfg.SSHPrivateKeyPath == "" {
		path := ssh_config.Get(cfg.SSHHost, "IdentityFile")
		if strings.HasPrefix(path, "~/") {
			d, err := os.UserHomeDir()
			if err != nil {
				return Config{}, err
			}

			path = strings.Replace(path, "~", d, 1)
		}
		cfg.SSHPrivateKeyPath = path
	}

	return cfg, nil
}

func (cfg Config) SSHURL() string {
	return "ssh://" + cfg.SSHUser + "@" + cfg.SSHHost + ":" + cfg.SSHPort
}
