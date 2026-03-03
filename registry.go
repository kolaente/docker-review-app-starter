package main

import (
	"encoding/json"
	"fmt"
	"log"
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
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repo, ref.Tag)
	log.Printf("registry: checking %s/%s:%s (url: %s)", ref.Registry, ref.Repo, ref.Tag, manifestURL)

	resp, err := rc.doRegistryRequest(manifestURL, "")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle auth challenge: registry returns 401 with Www-Authenticate header
	// containing a token endpoint. Fetch a bearer token and retry.
	if resp.StatusCode == http.StatusUnauthorized {
		token, err := rc.fetchToken(resp)
		if err != nil {
			log.Printf("registry: %s/%s:%s failed to obtain token: %v", ref.Registry, ref.Repo, ref.Tag, err)
			return "", nil
		}
		_ = resp.Body.Close()

		resp, err = rc.doRegistryRequest(manifestURL, token)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("registry: %s/%s:%s not found (404)", ref.Registry, ref.Repo, ref.Tag)
		return "", nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		log.Printf("registry: %s/%s:%s unauthorized even after token exchange (401)", ref.Registry, ref.Repo, ref.Tag)
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d for %s/%s:%s", resp.StatusCode, ref.Registry, ref.Repo, ref.Tag)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	log.Printf("registry: %s/%s:%s exists (digest: %s)", ref.Registry, ref.Repo, ref.Tag, digest)
	return digest, nil
}

func (rc *RegistryClient) doRegistryRequest(url, token string) (*http.Response, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry request failed: %w", err)
	}
	return resp, nil
}

// fetchToken parses the Www-Authenticate header from a 401 response and fetches a bearer token.
// Supports the format: Bearer realm="https://...",service="...",scope="..."
func (rc *RegistryClient) fetchToken(resp *http.Response) (string, error) {
	authHeader := resp.Header.Get("Www-Authenticate")
	if authHeader == "" {
		return "", fmt.Errorf("no Www-Authenticate header in 401 response")
	}

	params := parseWWWAuthenticate(authHeader)
	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("no realm in Www-Authenticate header: %s", authHeader)
	}

	// Build token request URL
	tokenURL := realm
	sep := "?"
	if service := params["service"]; service != "" {
		tokenURL += sep + "service=" + service
		sep = "&"
	}
	if scope := params["scope"]; scope != "" {
		tokenURL += sep + "scope=" + scope
	}

	log.Printf("registry: fetching token from %s", tokenURL)
	tokenResp, err := rc.HTTPClient.Get(tokenURL)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = tokenResp.Body.Close() }()

	if tokenResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint returned status %d", tokenResp.StatusCode)
	}

	var tokenBody struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenBody); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	token := tokenBody.Token
	if token == "" {
		token = tokenBody.AccessToken
	}
	if token == "" {
		return "", fmt.Errorf("empty token in response")
	}
	return token, nil
}

// parseWWWAuthenticate parses a Www-Authenticate header like:
// Bearer realm="https://ghcr.io/token",service="ghcr.io",scope="repository:org/repo:pull"
func parseWWWAuthenticate(header string) map[string]string {
	params := make(map[string]string)
	// Strip "Bearer " prefix
	header = strings.TrimPrefix(header, "Bearer ")
	header = strings.TrimPrefix(header, "bearer ")

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		params[k] = strings.Trim(v, "\"")
	}
	return params
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
