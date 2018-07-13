package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Project string
	Assets  []string
}

func init() {
	exec.Command("go", "get", "-u", "github.com/dags-/bundler")
	exec.Command("go", "install", "github.com/dags-/bundler")
}

func Build(config *Config) (artifacts []string, e error) {
	c := exec.Command("bundler", config.Project)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	e = c.Run()
	if e != nil {
		return artifacts, e
	}
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		for _, rule := range config.Assets {
			if match, e := filepath.Match(rule, path); e == nil && match {
				artifacts = append(artifacts, path)
			}
		}
		return nil
	})
	return artifacts, nil
}
