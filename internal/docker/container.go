package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	config "github.com/vrealzhou/lambda-local/config/cmd"
	"github.com/vrealzhou/lambda-local/internal/template"
)

const (
	imageName = "vreal/lambda-local-go:latest"
)

// StartLambdaContainer starts lambda container
func StartLambdaContainer(ctx context.Context, cli *client.Client, functions map[string]template.Function) error {
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	innerPort := strconv.Itoa(config.Port())
	io.Copy(os.Stdout, out)
	p, err := nat.NewPort("tcp", innerPort)
	if err != nil {
		return err
	}
	usr, err := user.Current()
	if err != nil {
		return err
	}

	envMap := make(map[string]string)
	credFile := filepath.Join(usr.HomeDir, ".aws", "credentials")
	if _, err := os.Stat(credFile); !os.IsNotExist(err) {
		creds := credentials.NewSharedCredentials(credFile, config.Profile())
		value, err := creds.Get()
		if err != nil {
			return err
		}
		envMap["AWS_ACCESS_KEY_ID"] = value.AccessKeyID
		envMap["AWS_SECRET_ACCESS_KEY"] = value.SecretAccessKey
		if value.SessionToken != "" {
			envMap["AWS_SESSION_TOKEN"] = value.SessionToken
		}
	}
	if config.EnvFile() != "" {
		envMap["ENV_JSON"] = "true"
	}
	envMap["PORT"] = innerPort
	envMap["AWS_DEFAULT_REGION"] = config.AWSRegion()
	envMap["AWS_REGION"] = config.AWSRegion()
	envMap["DEBUG"] = viper.GetString("debug")

	env := make([]string, 0)
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}
	
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		AttachStdout: true,
		AttachStderr: true,
		Image:        imageName,
		ExposedPorts: nat.PortSet{p: {}},
		Env:          env,
		Cmd:          strslice.StrSlice{"/var/lambdas/main"},
	}, &container.HostConfig{
		NetworkMode: config.NetworkMode(),
		PortBindings: nat.PortMap{p: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: innerPort,
			},
		}},
		AutoRemove: true,
	}, nil, "lambda-local-go")
	if err != nil {
		return err
	}
	fmt.Println("containerID:", resp.ID)
	config.SetContainerID(resp.ID)
	if err := copyToContainer(ctx, cli, "/var/lambdas/", config.Template()); err != nil {
		return err
	}
	for _, f := range functions {
		if err := copyToContainer(ctx, cli, "/var/lambdas/", f.Properties.CodeURI); err != nil {
			return err
		}
	}
	if config.EnvFile() != "" {
		err = copyToContainer(ctx, cli, "/var/lambdas/env.json", config.EnvFile())
		if err != nil {
			return err
		}
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	return nil
}

// StopContainer stops lambda container
func StopContainer(ctx context.Context, cli *client.Client) error {
	if err := cli.ContainerKill(ctx, config.ContainerID(), ""); err != nil {
		return err
	}
	return nil
}

// DeleteContainer deletes lambda container
func DeleteContainer(ctx context.Context, cli *client.Client) error {
	if err := cli.ContainerRemove(ctx, config.ContainerID(), types.ContainerRemoveOptions{}); err != nil {
		return err
	}
	return nil
}

func copyToContainer(ctx context.Context, cli *client.Client, dstPath, filename string) error {
	args := []string{"cp", filename, config.ContainerID() + ":" + dstPath}
	log.Debugf("Command: docker %s\n", args)
	cmd := exec.Command("docker", args...)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// AttachContainer attaches lambda container
func AttachContainer(ctx context.Context, cli *client.Client) error {
	hj, err := cli.ContainerAttach(ctx, config.ContainerID(), types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return err
	}
	buf := make([]byte, 4096)
	for {
		n, err := hj.Reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		os.Stdout.Write(buf[:n])
	}
}
