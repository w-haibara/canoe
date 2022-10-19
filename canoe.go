package canoe

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func Deploy() error {
	args := os.Args
	if len(args) < 3 {
		return fmt.Errorf("args missing")
	}
	sshHost := args[1]
	image := args[2]
	p := args[3]
	port, err := strconv.Atoi(p)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig(sshHost)
	if err != nil {
		return err
	}

	cli, err := NewClient(cfg)
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()

	fmt.Println("--- copy image start ---")
	if err := cli.CopyDockerImageToRemote(ctx, image); err != nil {
		return err
	}
	fmt.Println("--- copy image end ---")

	containerID, err := cli.GetRemoteContainerByPort(ctx, port)
	if err != nil {
		return err
	}

	fmt.Println("existed container id:", containerID)

	if containerID != "" {
		fmt.Println("--- stop container start ---")
		if err := cli.StopRemoteContainer(ctx, containerID); err != nil {
			return err
		}
		fmt.Println("--- stop container end ---")
	}

	fmt.Println("--- start container start ---")
	imageName := strings.Split(image, ":")[0]
	if err := cli.StartRemoteContainer(ctx, imageName, "NewContainer", 80, port); err != nil {
		return err
	}
	fmt.Println("--- start container end ---")

	return nil
}
