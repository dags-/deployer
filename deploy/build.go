package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Project struct {
	Owner  string
	Name   string
	Assets []string
}

func init() {
	exec.Command("go", "get", "-u", "github.com/dags-/bundler")
	exec.Command("go", "install", "github.com/dags-/bundler")
}

func Build(project *Project) (artifacts []string, e error) {
	p := fmt.Sprintf("github.com/%s/%s", project.Owner, project.Name)

	c := exec.Command("bundler", p)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	e = c.Run()
	if e != nil {
		return artifacts, e
	}

	for i, r := range project.Assets {
		project.Assets[i] = filepath.FromSlash(r)
	}

	dir := workDir(p)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		rel, _ := filepath.Rel(dir, path)
		for _, rule := range project.Assets {
			if match, e := filepath.Match(rule, rel); e == nil && match {
				artifacts = append(artifacts, path)
			}
		}
		return nil
	})

	return artifacts, nil
}

func workDir(path string) string {
	if _, e := os.Stat(path); e == nil {
		return path
	}

	path = filepath.Join(os.Getenv("GOPATH"), "src", path)
	if _, e := os.Stat(path); e == nil {
		return path
	}

	return "."
}
