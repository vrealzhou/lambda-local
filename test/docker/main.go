package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	imageName := "vreal/lambda-local-go:latest"

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	innerPort := "3001"
	io.Copy(os.Stdout, out)
	port, err := nat.NewPort("tcp", innerPort)
	if err != nil {
		panic(err)
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		AttachStdout: true,
		AttachStderr: true,
		Image:        imageName,
		ExposedPorts: nat.PortSet{port: {}},
		Env: []string{
			"PORT=" + innerPort,
		},
		Cmd: strslice.StrSlice{"/var/lambdas/main"},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{port: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: innerPort,
			},
		}},
		AutoRemove: true,
	}, nil, "lambda-local-go")
	if err != nil {
		panic(err)
	}
	fmt.Println("containerID:", resp.ID)
	if err := copyToContainer(cli, ctx, resp.ID, "/var/lambdas/", "deployments/ingestor-sam.yaml"); err != nil {
		panic(err)
	}
	if err := copyToContainer(cli, ctx, resp.ID, "/var/lambdas/", "build/lambdas/Hello.zip"); err != nil {
		panic(err)
	}
	if err := copyToContainer(cli, ctx, resp.ID, "/var/lambdas/", "build/lambdas/Cheers.zip"); err != nil {
		panic(err)
	}
	// if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
	// 	panic(err)
	// }
}

func copyToContainer(cli *client.Client, ctx context.Context, containerID, dstPath, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := cli.CopyToContainer(ctx, containerID, dstPath, file, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	}); err != nil {
		return err
	}
	return nil
}
