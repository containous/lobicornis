package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/google/go-github/v32/github"
	"github.com/rs/zerolog/log"
	"github.com/traefik/lobicornis/v2/pkg/conf"
	"github.com/traefik/lobicornis/v2/pkg/logger"
	"github.com/traefik/lobicornis/v2/pkg/repository"
	"github.com/traefik/lobicornis/v2/pkg/search"
	"golang.org/x/oauth2"
)

func main() {
	filename := flag.String("config", "./lobicornis.yml", "Path to the configuration file.")
	serverMode := flag.Bool("server", false, "Run as a web server.")
	version := flag.Bool("version", false, "Display version information.")
	help := flag.Bool("h", false, "Show this help.")

	flag.Usage = usage
	flag.Parse()
	if *help {
		usage()
		return
	}

	nArgs := flag.NArg()
	if nArgs > 0 {
		usage()
		return
	}

	if version != nil && *version {
		displayVersion()
		return
	}

	if filename == nil || *filename == "" {
		usage()
		return
	}

	cfg, err := conf.Load(*filename)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load config")
	}

	logLevel := cfg.Extra.LogLevel
	if cfg.Extra.DryRun {
		logLevel = "debug"
	}
	logger.Setup(logLevel)
	if *serverMode {
		err = launch(cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to launch the server")
		}
	} else {
		err = run(cfg)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to run the command")
		}
	}
}

func launch(cfg conf.Configuration) error {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			log.Error().Str("method", req.Method).Msg("Invalid http method")
			http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		err := run(cfg)
		if err != nil {
			log.Err(err).Msg("Report error")
			http.Error(rw, "Report error.", http.StatusInternalServerError)
			return
		}

		_, err = fmt.Fprint(rw, "Myrmica Lobicornis: Scheduled.\n")
		if err != nil {
			log.Err(err).Msg("Report error")
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return http.ListenAndServe(":"+strconv.Itoa(cfg.Server.Port), handler)
}

func run(cfg conf.Configuration) error {
	ctx := context.Background()

	client := newGitHubClient(ctx, cfg.Github.Token, cfg.Github.URL)

	finder := search.New(client, cfg.Markers, cfg.Retry)

	// search PRs with the FF merge method.
	ffResults, err := finder.Search(ctx, cfg.Github.User,
		search.WithLabels(cfg.Markers.MergeMethodPrefix+conf.MergeMethodFastForward),
		search.WithExcludedLabels(cfg.Markers.NoMerge, cfg.Markers.NeedMerge))
	if err != nil {
		return err
	}

	// search NeedMerge
	results, err := finder.Search(ctx, cfg.Github.User,
		search.WithLabels(cfg.Markers.NeedMerge),
		search.WithExcludedLabels(cfg.Markers.NeedHumanMerge, cfg.Markers.NoMerge))
	if err != nil {
		return err
	}

	for fullName, issues := range results {
		log.Info().Msgf("Repository %s", fullName)

		if _, ok := ffResults[fullName]; ok {
			log.Info().Msgf("Waiting for the merge of pull request with the label: %s", cfg.Markers.MergeMethodPrefix+conf.MergeMethodFastForward)
			continue
		}

		repoConfig := getRepoConfig(cfg, fullName)

		issue, err := finder.GetCurrentPull(issues)
		if err != nil {
			log.Err(err).Msg("unable to get the current pull request")
			continue
		}

		if issue == nil {
			log.Debug().Msgf("PR #%d: Nothing to merge.", issue.GetNumber())
			continue
		}

		repo := repository.New(client, fullName, cfg.Github.Token, cfg.Markers, cfg.Retry, cfg.Git, repoConfig, cfg.Extra)

		err = repo.Process(ctx, issue.GetNumber())
		if err != nil {
			log.Err(err).Msgf("PR #%d", issue.GetNumber())
		}
	}

	return nil
}

// newGitHubClient create a new GitHub client.
func newGitHubClient(ctx context.Context, token string, gitHubURL string) *github.Client {
	var tc *http.Client

	if len(token) != 0 {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(tc)

	if gitHubURL != "" {
		baseURL, err := url.Parse(gitHubURL)
		if err == nil {
			client.BaseURL = baseURL
		}
	}

	return client
}

func getRepoConfig(cfg conf.Configuration, repoName string) conf.RepoConfig {
	if repoCfg, ok := cfg.Repositories[repoName]; ok && repoCfg != nil {
		return *repoCfg
	}

	return cfg.Default
}

func usage() {
	_, _ = os.Stderr.WriteString("Myrmica Lobicornis:\n")
	flag.PrintDefaults()
}
