package menubar

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type defaultOpener struct{}

func NewOpener() opener {
	return &defaultOpener{}
}

func (o *defaultOpener) OpenURL(ctx context.Context, target string) error {
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("parse open url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("open url must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("open url host is required")
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("open url is only supported on macOS")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	openCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := exec.CommandContext(openCtx, "open", target).Run(); err != nil {
		return fmt.Errorf("open %q: %w", target, err)
	}
	return nil
}
