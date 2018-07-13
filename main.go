package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dags-/deployer/deploy"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

var (
	token    string
	secret   string
	port     string
	queue    chan *build
	projects map[string]*deploy.Project
)

type build struct {
	owner string
	repo  string
	id    int64
}

func init() {
	t := flag.String("token", "", "github token")
	s := flag.String("secret", "", "webhook secret")
	p := flag.String("port", "8095", "server port")
	flag.Parse()

	secret = mustFlag("secret", s)
	token = mustFlag("token", t)
	port = mustFlag("port", p)
	queue = make(chan *build)
	projects = deploy.LoadProjects()
}

func main() {
	go handleBuilds()
	go handleCommands()
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(handleRelease, github.ReleaseEvent)
	e := webhooks.Run(hook, ":"+port, "/webhooks")
	if e != nil {
		panic(e)
	}
}

func handleCommands() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := strings.ToLower(strings.TrimSpace(s.Text()))
		if line == "stop" {
			log.Println("stop invoked")
			os.Exit(0)
			continue
		}

		if strings.HasPrefix(line, "build") {
			split := strings.Split(line, " ")
			if len(split) != 2 {
				log.Println("owner/repo required")
				continue
			}

			project, exists := projects[split[1]]
			if !exists {
				log.Println("invalid project", split[1])
				continue
			}

			b, e := latestRelease(project.Owner, project.Name)
			if e != nil {
				log.Println("get release error:", e)
			} else {
				log.Println("build invoked")
				queue <- b
			}
		}
	}
}

func handleRelease(payload interface{}, header webhooks.Header) {
	r := payload.(github.ReleasePayload)
	if r.Release.Draft || r.Release.Prerelease {
		return
	}

	repo := r.Repository.Owner.Login + "/" + r.Repository.Name
	log.Println("release received:", repo)
	queue <- &build{
		owner: r.Repository.Owner.Login,
		repo:  r.Repository.Name,
		id:    r.Release.ID,
	}
}

func handleBuilds() {
	for b := range queue {
		project, exists := projects[b.owner+"/"+b.repo]
		if !exists {
			continue
		}

		log.Println("build received")
		artifacts, e := deploy.Build(project)
		if e != nil {
			log.Println("deploy error:", e)
			return
		}

		log.Printf("uploading %v artifacts\n", len(artifacts))
		for _, artifact := range artifacts {
			e := deploy.UploadAsset(b.owner, b.repo, b.id, artifact, token)
			if e != nil {
				log.Println("upload error:", e)
			} else {
				log.Println("upload complete:", artifact)
			}
			time.Sleep(time.Second)
		}
		log.Println("deploy complete!")
	}
}

func latestRelease(owner, repo string) (*build, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	r, e := http.Get(url)
	if e != nil {
		return nil, e
	}
	defer r.Body.Close()

	var rel struct {
		ID int64 `json:"id"`
	}

	e = json.NewDecoder(r.Body).Decode(&rel)
	if e != nil {
		return nil, e
	}

	return &build{owner: owner, repo: repo, id: rel.ID}, nil
}

func mustFlag(name string, val *string) string {
	if val == nil || *val == "" {
		panic("missing flag: " + name)
	}
	return *val
}
