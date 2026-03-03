package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const projectPrefix = "review-"

// RenderTemplate reads the compose template, replaces ${SUBDOMAIN}, writes to a temp file.
// Returns the path to the rendered file and a cleanup function.
func RenderTemplate(templatePath, subdomain string) (string, func(), error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", nil, fmt.Errorf("reading template: %w", err)
	}

	rendered := strings.ReplaceAll(string(data), "${SUBDOMAIN}", subdomain)

	tmpfile, err := os.CreateTemp("", fmt.Sprintf("compose-%s-*.yml", subdomain))
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := tmpfile.WriteString(rendered); err != nil {
		os.Remove(tmpfile.Name())
		return "", nil, fmt.Errorf("writing rendered template: %w", err)
	}
	tmpfile.Close()

	cleanup := func() { os.Remove(tmpfile.Name()) }
	return tmpfile.Name(), cleanup, nil
}

func projectName(subdomain string) string {
	return projectPrefix + subdomain
}

type ComposeManager struct {
	TemplatePath string
}

func (cm *ComposeManager) StartStack(subdomain string) error {
	composePath, cleanup, err := RenderTemplate(cm.TemplatePath, subdomain)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.Command("docker", "compose", "-p", projectName(subdomain), "-f", composePath, "up", "-d", "--wait")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cm *ComposeManager) StopStack(subdomain string) error {
	cmd := exec.Command("docker", "compose", "-p", projectName(subdomain), "down", "--remove-orphans", "--volumes")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cm *ComposeManager) PullAndRestart(subdomain string) error {
	composePath, cleanup, err := RenderTemplate(cm.TemplatePath, subdomain)
	if err != nil {
		return err
	}
	defer cleanup()

	pull := exec.Command("docker", "compose", "-p", projectName(subdomain), "-f", composePath, "pull")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	up := exec.Command("docker", "compose", "-p", projectName(subdomain), "-f", composePath, "up", "-d")
	up.Stdout = os.Stdout
	up.Stderr = os.Stderr
	return up.Run()
}

// ListRunningStacks returns subdomains of all review-* compose projects currently running.
func ListRunningStacks() ([]string, error) {
	cmd := exec.Command("docker", "compose", "ls", "--format", "json", "--filter", "name="+projectPrefix)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing compose projects: %w", err)
	}

	// Parse JSON output to extract project names and strip prefix
	var projects []struct {
		Name string `json:"Name"`
	}
	if err := json.Unmarshal(out, &projects); err != nil {
		return nil, fmt.Errorf("parsing compose ls output: %w", err)
	}

	var subdomains []string
	for _, p := range projects {
		sub := strings.TrimPrefix(p.Name, projectPrefix)
		if sub != "" {
			subdomains = append(subdomains, sub)
		}
	}
	return subdomains, nil
}
