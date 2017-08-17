package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/containous/flaeg"
	"github.com/containous/lobicornis/core"
	"github.com/containous/lobicornis/gh"
	"github.com/ldez/go-git-cmd-wrapper/config"
	"github.com/ldez/go-git-cmd-wrapper/git"
)

func main() {
	config := &core.Configuration{
		MinReview:          1,
		DryRun:             true,
		DefaultMergeMethod: gh.MergeMethodSquash,
		MergeMethodPrefix:  "bot/merge-method-",
		LabelMarkers: &core.LabelMarkers{
			NeedHumanMerge:  "bot/need-human-merge",
			NeedMerge:       "status/3-needs-merge",
			MergeInProgress: "status/4-merge-in-progress",
		},
		ForceNeedUpToDate: true,
		ServerPort:        80,
	}

	defaultPointersConfig := &core.Configuration{LabelMarkers: &core.LabelMarkers{}}
	rootCmd := &flaeg.Command{
		Name:                  "lobicornis",
		Description:           `Myrmica Lobicornis:  Update and Merge Pull Request from GitHub.`,
		Config:                config,
		DefaultPointersConfig: defaultPointersConfig,
		Run: func() error {
			if config.Debug {
				log.Printf("Run Lobicornis command with config : %+v\n", config)
			}

			if config.DryRun {
				log.Print("IMPORTANT: you are using the dry-run mode. Use `--dry-run=false` to disable this mode.")
			}

			if len(config.GitHubToken) == 0 {
				config.GitHubToken = os.Getenv("GITHUB_TOKEN")
			}

			required(config.GitHubToken, "token")
			required(config.Owner, "owner")
			required(config.RepositoryName, "repo-name")
			required(config.DefaultMergeMethod, "merge-method")

			required(config.LabelMarkers.NeedMerge, "need-merge")
			required(config.LabelMarkers.MergeInProgress, "merge-in-progress")
			required(config.LabelMarkers.NeedHumanMerge, "need-human-merge")

			err := launch(config)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	flag := flaeg.New(rootCmd, os.Args[1:])
	flag.Run()
}

func launch(config *core.Configuration) error {
	err := configureGitUserInfo(config.GitUserName, config.GitUserEmail)
	if err != nil {
		return err
	}

	if config.ServerMode {
		server := &server{config: config}
		return server.ListenAndServe()
	}

	return core.Execute(*config)
}

func configureGitUserInfo(gitUserName string, gitUserEmail string) error {
	if len(gitUserEmail) != 0 {
		output, err := git.Config(config.Entry("user.email", gitUserEmail))
		if err != nil {
			log.Println(output)
			return err
		}
	}

	if len(gitUserName) != 0 {
		output, err := git.Config(config.Entry("user.name", gitUserName))
		if err != nil {
			log.Println(output)
			return err
		}
	}

	return nil
}

func required(field string, fieldName string) error {
	if len(field) == 0 {
		log.Fatalf("%s is mandatory.", fieldName)
	}
	return nil
}

type server struct {
	config *core.Configuration
}

func (s *server) ListenAndServe() error {
	return http.ListenAndServe(":"+strconv.Itoa(s.config.ServerPort), s)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Printf("Invalid http method: %s", r.Method)
		http.Error(w, "405 Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := core.Execute(*s.config)
	if err != nil {
		log.Printf("Report error: %v", err)
		http.Error(w, "Report error.", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Scheluded.")
}
