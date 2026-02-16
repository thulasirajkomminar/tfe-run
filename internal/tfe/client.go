// Package tfe provides utilities for interacting with the Terraform Enterprise (TFE) API, including client creation and token resolution.
package tfe

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/go-tfe"
	log "github.com/sirupsen/logrus"
)

type credentialsTFRCJSON struct {
	Credentials map[string]struct {
		Token string `json:"token"`
	} `json:"credentials"`
}

// NewClient creates a new TFE client by resolving the token from:
// 1. TFE_TOKEN environment variable
// 2. ~/.terraformrc
// 3. ~/.terraform.d/credentials.tfrc.json.
func NewClient() (*tfe.Client, error) {
	token, err := resolveToken()
	if err != nil {
		return nil, fmt.Errorf("unable to resolve TFE token: %w", err)
	}

	client, err := tfe.NewClient(&tfe.Config{Token: token})
	if err != nil {
		return nil, fmt.Errorf("unable to create TFE client: %w", err)
	}

	return client, nil
}

func resolveToken() (string, error) {
	if token := os.Getenv("TFE_TOKEN"); token != "" {
		log.Info("Using token from TFE_TOKEN environment variable")
		return token, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine home directory: %w", err)
	}

	token, err := readTerraformRC(homeDir)
	if err == nil && token != "" {
		log.Info("Using token from ~/.terraformrc")
		return token, nil
	}

	token, err = readCredentialsTFRC(homeDir)
	if err == nil && token != "" {
		log.Info("Using token from ~/.terraform.d/credentials.tfrc.json")
		return token, nil
	}

	return "", errors.New("no TFE token found. Set TFE_TOKEN env var, or configure ~/.terraformrc or ~/.terraform.d/credentials.tfrc.json")
}

func readTerraformRC(homeDir string) (string, error) {
	path := filepath.Clean(filepath.Join(homeDir, ".terraformrc"))

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`token\s*=\s*"(.*)"`)
	match := re.FindStringSubmatch(string(content))

	if match == nil || len(match) != 2 {
		return "", fmt.Errorf("no token found in %s", path)
	}

	return match[1], nil
}

func readCredentialsTFRC(homeDir string) (string, error) {
	path := filepath.Clean(filepath.Join(homeDir, ".terraform.d", "credentials.tfrc.json"))

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var creds credentialsTFRCJSON

	err = json.Unmarshal(content, &creds)
	if err != nil {
		re := regexp.MustCompile(`"token"\s*:\s*"(.*)"`)

		match := re.FindStringSubmatch(string(content))
		if match == nil || len(match) != 2 {
			return "", fmt.Errorf("no token found in %s", path)
		}

		return match[1], nil
	}

	for host, cred := range creds.Credentials {
		if cred.Token != "" {
			log.Infof("Found token for host: %s", host)
			return cred.Token, nil
		}
	}

	return "", fmt.Errorf("no token found in %s", path)
}
