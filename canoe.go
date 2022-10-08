package canoe

import (
	"context"
)

func Deploy() error {
	cfg, err := LoadConfig(Config{
		SSHHost: "app.w-haibara.com",
	})
	if err != nil {
		return err
	}

	cli, err := NewClient(cfg)
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()

	imageID := "myapp"
	if err := cli.CopyDockerImageToRemote(ctx, imageID); err != nil {
		return err
	}

	return nil
}
