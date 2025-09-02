package monitor

import (
	"fmt"
	"os"
	"strings"
)

func IsRunningInDocker() bool {
	// Check for /.dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check /proc/1/cgroup for docker
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, "docker") || strings.Contains(content, "/lxc/")
}

func IsUbuntu() bool {
	// Check /etc/os-release for Ubuntu
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}

	content := strings.ToLower(string(data))
	return strings.Contains(content, "ubuntu")
}

func ValidateEnvironment() error {
	if !IsRunningInDocker() {
		return fmt.Errorf("application must run inside Docker container")
	}

	if !IsUbuntu() {
		return fmt.Errorf("application must run on Ubuntu base image")
	}

	return nil
}
