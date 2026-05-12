package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/mochaka/devproxy/internal/discovery"
)

func DefaultDockerPing(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker info failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func DefaultDockerScan(ctx context.Context) ([]ContainerState, error) {
	ids, err := listRunningContainerIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}

	args := append([]string{"inspect"}, ids...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var containers []dockerInspectContainer
	if err := json.Unmarshal(out, &containers); err != nil {
		return nil, fmt.Errorf("decode docker inspect output: %w", err)
	}

	states := make([]ContainerState, 0, len(containers))
	for _, container := range containers {
		states = append(states, ContainerState{
			ID:      container.ID,
			Name:    container.Name,
			Running: container.State.Running,
			Labels:  container.Config.Labels,
			Ports:   container.publishedPorts(),
		})
	}
	return states, nil
}

func DefaultDockerEvents(ctx context.Context) (*EventStream, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	args := []string{"events", "--format", "{{json .}}", "--filter", "type=container"}
	for _, action := range supportedDockerEvents {
		args = append(args, "--filter", "event="+action)
	}
	cmd := exec.CommandContext(streamCtx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("prepare docker events output: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start docker events: %w", err)
	}

	events := make(chan DockerEvent)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if streamCtx.Err() != nil {
				return
			}
			var payload dockerEventMessage
			if err := json.Unmarshal(scanner.Bytes(), &payload); err != nil {
				errs <- fmt.Errorf("decode docker events output: %w", err)
				cancel()
				_ = cmd.Wait()
				return
			}
			action := strings.TrimSpace(payload.Action)
			if action == "" {
				action = strings.TrimSpace(payload.Status)
			}
			if action == "" {
				continue
			}
			events <- DockerEvent{Action: action}
		}
		if err := scanner.Err(); err != nil {
			if streamCtx.Err() == nil {
				errs <- fmt.Errorf("read docker events output: %w", err)
			}
			cancel()
			_ = cmd.Wait()
			return
		}
		if err := cmd.Wait(); err != nil {
			if streamCtx.Err() != nil {
				return
			}
			msg := strings.TrimSpace(stderr.String())
			if msg != "" {
				errs <- fmt.Errorf("docker events failed: %w: %s", err, msg)
			} else {
				errs <- fmt.Errorf("docker events failed: %w", err)
			}
			return
		}
		if streamCtx.Err() == nil {
			errs <- io.EOF
		}
	}()

	return &EventStream{
		Events: events,
		Errors: errs,
		close: func() error {
			cancel()
			return nil
		},
	}, nil
}

func DefaultEnsureMKCert(context.Context) error {
	_, err := exec.LookPath("mkcert")
	if err != nil {
		return fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
	}
	return nil
}

type dockerInspectContainer struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

type dockerEventMessage struct {
	Status string `json:"status"`
	Action string `json:"Action"`
}

func (c dockerInspectContainer) publishedPorts() []discovery.PublishedPort {
	ports := make([]discovery.PublishedPort, 0)
	for spec, bindings := range c.NetworkSettings.Ports {
		_, protocol, ok := strings.Cut(spec, "/")
		if !ok || len(bindings) == 0 {
			continue
		}
		for _, binding := range bindings {
			hostPort, err := strconv.Atoi(strings.TrimSpace(binding.HostPort))
			if err != nil || hostPort <= 0 {
				continue
			}
			ports = append(ports, discovery.PublishedPort{
				HostIP:   binding.HostIP,
				HostPort: hostPort,
				Protocol: strings.ToLower(protocol),
			})
		}
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].HostPort == ports[j].HostPort {
			return ports[i].Protocol < ports[j].Protocol
		}
		return ports[i].HostPort < ports[j].HostPort
	})
	return ports
}

func listRunningContainerIDs(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.ID}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		id := strings.TrimSpace(line)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
