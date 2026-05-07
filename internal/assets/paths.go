package assets

import (
	"encoding/base64"
	"os"
	"path/filepath"
)

const (
	defaultWebDir   = "web/dist"
	defaultAgentDir = "bin/agents"
	defaultLogoPath = "web/public/logo.png"
)

func WebDir() string {
	if dir := os.Getenv("PIKA_WEB_DIR"); dir != "" {
		return dir
	}
	return defaultWebDir
}

func AgentDir() string {
	if dir := os.Getenv("PIKA_AGENT_DIR"); dir != "" {
		return dir
	}
	return defaultAgentDir
}

func AgentPath(filename string) string {
	return filepath.Join(AgentDir(), filename)
}

func DefaultLogoBase64() string {
	for _, path := range []string{
		os.Getenv("PIKA_DEFAULT_LOGO_PATH"),
		defaultLogoPath,
		filepath.Join(WebDir(), "logo.png"),
	} {
		if path == "" {
			continue
		}
		logo, err := os.ReadFile(path)
		if err == nil && len(logo) > 0 {
			return "data:image/png;base64," + base64.StdEncoding.EncodeToString(logo)
		}
	}
	return ""
}
