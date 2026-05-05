package certs

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type RunFunc func(binary string, args ...string) error

type MKCertIssuer struct {
	Binary string
	Runner RunFunc
}

type IssuedCertificate struct {
	ProjectRoot string
	SANs        []string
	CertPath    string
	KeyPath     string
}

func (i MKCertIssuer) Issue(outputDir string, sans []string) (IssuedCertificate, error) {
	if len(sans) == 0 {
		return IssuedCertificate{}, fmt.Errorf("mkcert issuance failed: no SANs provided")
	}

	primary := normalizeHostname(sans[0])
	if primary == "" {
		return IssuedCertificate{}, fmt.Errorf("mkcert issuance failed: invalid primary SAN")
	}

	binary := i.Binary
	if binary == "" {
		binary = "mkcert"
	}
	runner := i.Runner
	if runner == nil {
		runner = runCommand
	}

	certPath := filepath.Join(outputDir, sanitizeFilename(primary)+".pem")
	keyPath := filepath.Join(outputDir, sanitizeFilename(primary)+"-key.pem")
	args := []string{"-cert-file", certPath, "-key-file", keyPath}
	args = append(args, sans...)

	if err := runner(binary, args...); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return IssuedCertificate{}, fmt.Errorf("mkcert not found: install mkcert before enabling HTTPS: %w", err)
		}
		return IssuedCertificate{}, fmt.Errorf("mkcert issuance failed for %v: %w", sans, err)
	}

	return IssuedCertificate{ProjectRoot: primary, SANs: append([]string(nil), sans...), CertPath: certPath, KeyPath: keyPath}, nil
}

func runCommand(binary string, args ...string) error {
	cmd := exec.Command(binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return err
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func sanitizeFilename(host string) string {
	host = strings.ReplaceAll(host, "*", "wildcard")
	host = strings.ReplaceAll(host, "/", "-")
	return host
}
