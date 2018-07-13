package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dags-/deployer/deploy"
	"gopkg.in/go-playground/webhooks.v3"
	"gopkg.in/go-playground/webhooks.v3/github"
)

var (
	token  string
	secret string
	port   string
	config *deploy.Config
)

func init() {
	t := flag.String("token", "", "github token")
	s := flag.String("secret", "", "webhook secret")
	p := flag.String("port", "8088", "server port")
	flag.Parse()

	secret = mustFlag("secret", s)
	token = mustFlag("token", t)
	port = mustFlag("port", p)

	config = &deploy.Config{
		Project: "github.com/dags-/launch",
		Assets: []string{
			"builds/darwin/*.zip",
			"builds/windows/*.zip",
			"builds/linux/*.AppImage",
		},
	}
}

func main() {
	go handleStop()
	hook := github.New(&github.Config{Secret: secret})
	hook.RegisterEvents(release, github.ReleaseEvent)
	e := webhooks.Run(hook, ":"+port, "/webhooks")
	if e != nil {
		panic(e)
	}
}

func handleStop() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := strings.ToLower(s.Text())
		if strings.HasPrefix(line, "stop") {
			os.Exit(0)
		}
	}
}

func mustFlag(name string, val *string) string {
	if val == nil || *val == "" {
		panic("missing flag: " + name)
	}
	return *val
}

func release(payload interface{}, header webhooks.Header) {
	r := payload.(github.ReleasePayload)
	if r.Release.Draft || r.Release.Prerelease {
		return
	}

	log.Println("release received:", r.Repository.Name)
	artifacts, e := deploy.Build(config)
	if e != nil {
		log.Println("deploy error:", e)
		return
	}

	for _, artifact := range artifacts {
		e := deploy.UploadAsset("", r.Repository.Name, r.Release.ID, artifact, token)
		if e != nil {
			log.Println("upload error:", e)
		}
		time.Sleep(time.Second)
	}
}
