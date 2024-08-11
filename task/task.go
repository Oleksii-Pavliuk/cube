package task

import (
	"io"
	"log"
	"math"
	"os"
	"time"

	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/moby/moby/pkg/stdcopy"
)

type Task struct {
	ID            uuid.UUID
	ContainerID   string
	Name          string
	State         State
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime time.Time
	FinishTime time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}


type ContainerConfig struct {
	// Name of the task, also used as the container name
	Name string
	// AttachStdin boolean which determines if stdin should be attached
	AttachStdin bool
	// AttachStdout boolean which determines if stdout should be attached
	AttachStdout bool
	// AttachStderr boolean which determines if stderr should be attached
	AttachStderr bool
	// ExposedPorts list of ports exposed
	ExposedPorts nat.PortSet
	// Cmd to be run inside container (optional)
	Cmd []string
	// Image used to run the container
	Image string
	// Cpu
	Cpu float64
	// Memory in MiB
	Memory int64
	// Disk in GiB
	Disk int64
	// Env variables
	Env []string

	RestartPolicy string
}

func NewConfig(t*Task) *ContainerConfig {
	return &ContainerConfig{
		Name: t.Name,
		ExposedPorts: t.ExposedPorts,
		Image: t.Image,
		Cpu: t.Cpu,
		Memory: t.Memory,
		Disk: t.Disk,
		RestartPolicy: t.RestartPolicy,
	}
}



type Docker struct {
	Client *client.Client
	Config   ContainerConfig
}

func NewDocker(c *ContainerConfig) *Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)
	return &Docker{
		Client: dc,
		Config: *c,
	}
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerId string
	Result      string
}


func (docker *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := docker.Client.ImagePull(ctx, docker.Config.Image, image.PullOptions{})
	if err != nil {
			log.Printf("Error pulling image %s: %v\n", docker.Config.Image, err)
			return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	restartPolicy := container.RestartPolicy{
			Name: container.RestartPolicyMode(docker.Config.RestartPolicy),
	}

	resources := container.Resources{
			Memory: docker.Config.Memory,
			NanoCPUs: int64(docker.Config.Cpu * math.Pow(10, 9)),
	}

	containerConfig := container.Config{
			Image: docker.Config.Image,
			Tty: false,
			Env: docker.Config.Env,
			ExposedPorts: docker.Config.ExposedPorts,
	}

	hostConfig := container.HostConfig{
			RestartPolicy: restartPolicy,
			Resources:     resources,
			PublishAllPorts: true,
	}

	resp, err := docker.Client.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, docker.Config.Name)
	if err != nil {
			log.Printf("Error creating container using image %s: %v\n", docker.Config.Image, err)
			return DockerResult{Error: err}
	}

	err = docker.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
			log.Printf("Error starting container %s: %v\n", resp.ID, err)
			return DockerResult{Error: err}
	}

	out, err := docker.Client.ContainerLogs(ctx,resp.ID,container.LogsOptions{ShowStdout: true, ShowStderr: true})

	if err != nil {
			log.Printf("Error getting logs for container %s: %v\n", resp.ID, err)
			return DockerResult{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return DockerResult{ContainerId: resp.ID, Action: "start",Result: "success"}
}



func (docker *Docker) Stop(id string,remove bool) DockerResult {
	log.Printf("Attempting to stop container %v", id)
	ctx := context.Background()
	err := docker.Client.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
			log.Printf("Error stopping container %s: %v\n", id, err)
			return DockerResult{Error: err}
	}

	if remove {
		err = docker.Client.ContainerRemove(ctx, id, container.RemoveOptions{
			RemoveVolumes: true,
			RemoveLinks:   false,
			Force:         false,
		})
		if err != nil {
				log.Printf("Error removing container %s: %v\n", id, err)
				return DockerResult{Error: err}
		}
	}

	return DockerResult{Action: "stop", Result: "success", Error: nil}
}