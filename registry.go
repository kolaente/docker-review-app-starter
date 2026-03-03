package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ImageRef struct {
	Registry string
	Repo     string
	Tag      string
}

func ParseImageRef(image string) (*ImageRef, error) {
	// Split tag
	tag := "latest"
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		// Make sure this colon isn't part of the registry (port)
		rest := image[idx+1:]
		if !strings.Contains(rest, "/") {
			tag = rest
			image = image[:idx]
		}
	}

	// Split registry from repo
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("cannot parse image reference %q: no registry prefix", image)
	}

	return &ImageRef{
		Registry: parts[0],
		Repo:     parts[1],
		Tag:      tag,
	}, nil
}

type RegistryClient struct {
	HTTPClient *http.Client
}

// CheckTag queries the registry for a tag. Returns the digest if it exists, empty string if not.
func (rc *RegistryClient) CheckTag(ref *ImageRef) (string, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repo, ref.Tag)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json")

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	return digest, nil
}

func ParseTemplateImageRef(templatePath string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	// Find image lines containing ${SUBDOMAIN}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "image:") && strings.Contains(line, "${SUBDOMAIN}") {
			ref := strings.TrimPrefix(line, "image:")
			ref = strings.TrimSpace(ref)
			// Remove quotes if present
			ref = strings.Trim(ref, "\"'")
			return ref, nil
		}
	}

	return "", fmt.Errorf("no image with ${SUBDOMAIN} placeholder found in template")
}
