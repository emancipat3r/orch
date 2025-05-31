package utils

import (
	"io"
	"os"
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
