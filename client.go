package canoe

import (
	"context"
	"net/http"
	"os"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/client"
	docker "github.com/docker/docker/client"
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
	buf, err := os.ReadFile(cfg.SSHPrivateKeyPath)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            cfg.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
	}

	cli, err := ssh.Dial("tcp", cfg.SSHHost+":"+cfg.SSHPort, config)
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

func (c *Client) CopyDockerImageToRemote(ctx context.Context, imageID string) error {
	body, err := c.DockerLocal.ImageSave(ctx, []string{imageID})
	if err != nil {
		return err
	}
	defer body.Close()

	if _, err := c.DockerRemote.ImageLoad(ctx, body, true); err != nil {
		return err
	}

	return nil
}
