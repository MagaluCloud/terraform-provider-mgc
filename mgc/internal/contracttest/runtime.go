//go:build contract

package contracttest

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
)

// Run installs local-only API routing and requires a local Terraform CLI.
// Use os.Exit(contracttest.Run(m)) from each domain package TestMain.
func Run(m *testing.M) int {
	// Require an explicit local CLI, avoiding implicit downloads and skipped tests.
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" {
		path, err := exec.LookPath("terraform")
		if err != nil {
			fmt.Fprintln(os.Stderr, "contract tests require terraform on PATH or TF_ACC_TERRAFORM_PATH")
			return 1
		}
		os.Setenv("TF_ACC_TERRAFORM_PATH", path)
	}
	oldTransport, oldClient := http.DefaultTransport, http.DefaultClient
	defer func() { http.DefaultTransport, http.DefaultClient = oldTransport, oldClient }()
	http.DefaultTransport = network
	http.DefaultClient = &http.Client{Transport: network}
	code := m.Run()
	network.mu.RLock()
	blocked := append([]string(nil), network.blocked...)
	network.mu.RUnlock()
	if len(blocked) != 0 {
		fmt.Fprintf(os.Stderr, "unexpected outbound requests blocked: %v\n", blocked)
		code = 1
	}
	return code
}
