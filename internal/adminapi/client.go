package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/mochaka/devproxy/internal/admin"
)

type Client struct {
	socketPath string
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			if err != nil {
				return nil, fmt.Errorf("connect admin socket %q: %w", socketPath, err)
			}
			return conn, nil
		},
	}
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{Transport: transport, Timeout: 10 * time.Second},
	}
}

func (c *Client) Status(ctx context.Context) (admin.StatusView, error) {
	payload, err := fetchJSON[StatusResponse](ctx, c.httpClient, "/status")
	if err != nil {
		return admin.StatusView{}, err
	}
	return payload.Status, nil
}

func (c *Client) Routes(ctx context.Context) ([]admin.RouteView, error) {
	payload, err := fetchJSON[RoutesResponse](ctx, c.httpClient, "/routes")
	if err != nil {
		return nil, err
	}
	return payload.Routes, nil
}

func (c *Client) Logs(ctx context.Context) ([]admin.LogEvent, error) {
	payload, err := fetchJSON[LogsResponse](ctx, c.httpClient, "/logs")
	if err != nil {
		return nil, err
	}
	return payload.Events, nil
}

func (c *Client) Doctor(ctx context.Context) (admin.DoctorView, error) {
	payload, err := fetchJSON[DoctorResponse](ctx, c.httpClient, "/doctor")
	if err != nil {
		return admin.DoctorView{}, err
	}
	return payload.Doctor, nil
}

func (c *Client) Refresh(ctx context.Context, reason string) (RefreshResponse, error) {
	request := RefreshRequest{Reason: reason}
	body, err := json.Marshal(request)
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("encode refresh request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix/refresh", bytes.NewReader(body))
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("build /refresh request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return RefreshResponse{}, fmt.Errorf("request /refresh: %w", err)
	}
	defer resp.Body.Close()

	var payload RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return RefreshResponse{}, fmt.Errorf("decode /refresh response: %w", err)
	}
	return payload, nil
}

func fetchJSON[T any](ctx context.Context, client *http.Client, path string) (T, error) {
	var zero T
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return zero, fmt.Errorf("build %s request: %w", path, err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	var payload T
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return zero, fmt.Errorf("decode %s response: %w", path, err)
	}
	return payload, nil
}
