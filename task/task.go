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
	Name          string
	State         State
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBindings  map[string]string
	RestartPolicy string
	StartTime time.Time
	FinishTyme time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	Timestamp time.Time
	Task      Task
}

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)


type ContainerConfig struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	Cpu           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy container.RestartPolicyMode
}


type Docker struct {
	Client *client.Client
	Config   ContainerConfig
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
			Name: docker.Config.RestartPolicy,
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