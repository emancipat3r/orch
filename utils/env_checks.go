package utils

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

/*
	type Config struct {
		Key string 'toml:"key"'
	}

	type Provider struct {
		DigitalOcean	Config 'toml:"digitalocean"'
		Linode		Config 'toml:"linode"'
		Vultr		Config 'toml:"vultr"'
	}

var cfg Provider
_, err := toml.DecodeFile("provider_credentials.toml", &cfg)
*/

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		return filepath.Join(usr.HomeDir, strings.TrimPrefix(path, "~"))
	}
	return path
}

func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}
