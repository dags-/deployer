package deploy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

type Project struct {
	Owner  string   `json:"owner"`
	Name   string   `json:"name"`
	Assets []string `json:"assets"`
}

func init() {
	exec.Command("go", "get", "-u", "github.com/dags-/bundler")
	exec.Command("go", "install", "github.com/dags-/bundler")
}

func LoadProjects() map[string]*Project {
	u, e := user.Current()
	if e != nil {
		panic(e)
	}

	dir := filepath.Join(u.HomeDir, "deployer")
	if _, e := os.Stat(dir); e != nil {
		os.Mkdir(dir, os.ModePerm)
		return map[string]*Project{}
	}

	fs, e := ioutil.ReadDir(dir)
	if e != nil {
		panic(e)
	}

	projects := make(map[string]*Project)
	for _, f := range fs {
		o, e := os.Open(filepath.Join(dir, f.Name()))
		if e != nil {
			log.Println("open file error:", e)
			continue
		}

		var project Project
		e = json.NewDecoder(o).Decode(&project)
		if e != nil {
			log.Println("decode err:", e)
			continue
		}

		projects[project.Owner+"/"+project.Name] = &project
	}
	return projects
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
