package canoe

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	DockerLocal  *docker.Client
	DockerRemote *docker.Client
	SSH          *ssh.Client
	SFTP         *sftp.Client
}

func NewClient(cfg Config) (*Client, error) {
	dlocal, err := newDockerLocalClient()
	if err != nil {
		return nil, err
	}

	dremote, err := newDockerRemoteClient(cfg.SSHURL())
	if err != nil {
		return nil, err
	}

	ssh, err := newSSHClient(cfg)
	if err != nil {
		return nil, err
	}

	sftp, err := newSFTPClient(ssh)
	if err != nil {
		return nil, err
	}

	return &Client{
		DockerLocal:  dlocal,
		DockerRemote: dremote,
		SSH:          ssh,
		SFTP:         sftp,
	}, nil
}

func newDockerLocalClient() (*docker.Client, error) {
	cli, err := docker.NewClientWithOpts(
		docker.FromEnv,
		docker.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func newDockerRemoteClient(sshurl string) (*docker.Client, error) {
	helper, err := connhelper.GetConnectionHelper(sshurl)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: helper.Dialer,
		},
	}

	cli, err := docker.NewClientWithOpts(
		docker.FromEnv,
		docker.WithAPIVersionNegotiation(),
		client.WithHTTPClient(httpClient),
		client.WithHost(helper.Host),
		client.WithDialContext(helper.Dialer),
	)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func newSSHClient(cfg Config) (*ssh.Client, error) {
	buf, err := os.ReadFile(cfg.SSH.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            cfg.SSH.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	cli, err := ssh.Dial("tcp", cfg.SSH.Host+":"+cfg.SSH.Port, config)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func newSFTPClient(ssh *ssh.Client) (*sftp.Client, error) {
	cli, err := sftp.NewClient(ssh)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

func (c *Client) Close() {
	c.DockerLocal.Close()
	c.DockerRemote.Close()
	c.SFTP.Close()
	c.SSH.Close()
}

func (c *Client) GetRemoteContainerByPort(ctx context.Context, port int) (string, error) {
	containers, err := c.DockerRemote.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return "", err
	}

	for _, c := range containers {
		for _, p := range c.Ports {
			if (int)(p.PublicPort) == port {
				return c.ID, nil
			}
		}
	}

	return "", nil
}

func (c *Client) StopRemoteContainer(ctx context.Context, containerID string) error {
	return c.DockerRemote.ContainerStop(ctx, containerID, container.StopOptions{})
}

func (c *Client) StartRemoteContainer(ctx context.Context, imageName, containerName string, privatePort, publishPort int) error {
	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: strconv.Itoa(publishPort),
	}
	containerPort, err := nat.NewPort("tcp", strconv.Itoa(publishPort))
	if err != nil {
		log.Println(err)
		return err
	}
	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}
	config := container.Config{Image: imageName}
	hostConfig := container.HostConfig{PortBindings: portBinding}
	createRes, err := c.DockerRemote.ContainerCreate(ctx, &config, &hostConfig, nil, nil, containerName)
	if err != nil {
		log.Println(err)
		return err
	}

	if err := c.DockerRemote.ContainerStart(ctx, createRes.ID, types.ContainerStartOptions{}); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (c *Client) GetLocatImageID(ctx context.Context, image string) (string, error) {
	images, err := c.DockerRemote.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return "", err
	}

	for _, i := range images {
		for _, tag := range i.RepoTags {
			if tag == image {
				return strings.Split(i.ID, ":")[1], nil
			}
		}
	}

	return "", nil
}

func (c *Client) CopyDockerImageToRemote(ctx context.Context, imageID string) error {
	log.Println("CopyDockerImageToRemote:: imageID:", imageID)
	body, err := c.DockerLocal.ImageSave(ctx, []string{imageID})
	if err != nil {
		log.Println(err)
		return err
	}
	defer body.Close()

	if _, err := c.DockerRemote.ImageLoad(ctx, body, true); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
