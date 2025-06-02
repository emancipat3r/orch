package utils

import (
	"io"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/emancipat3r/vps3/logger"
)

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	return true
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

func MakeDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

type Config struct {
	Key string `toml:"key"`
}

type Provider struct {
	DigitalOcean	Config 'toml:"digitalocean"'
	Linode			Config 'toml:"linode"'
	Vultr			Config 'toml:"vultr"'
}

var cfg Provider

func ParseCreds(path string, choice string) string {
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		logger.Error("Error parsing credentials file at %s: %v", path, err)
	}

	choice = strings.ToLower(choice)

	var key string
	switch choice {
	case "digitalocean":
		key = cfg.DigitalOcean.Key
	case "linode":
		key = cfg.Linode.Key
	case "vultr":
		key = cfg.Vultr.Key
	default:
		logger.Error("Invalid provider choice: %s", choice)
		return ""
	}

	if key == "" {
		logger.Error("Missing %s key in credentials file", choice)
		return ""
	}

	return key
}
}
