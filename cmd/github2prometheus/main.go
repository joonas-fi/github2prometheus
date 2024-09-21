package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/aws/lambdautils"
	"github.com/function61/gokit/httputils"
	"github.com/function61/gokit/logex"
	"github.com/function61/gokit/ossignal"
	"github.com/function61/gokit/promconstmetrics"
	"github.com/function61/gokit/taskrunner"
	"github.com/function61/prompipe/pkg/prompipeclient"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	promContentType = "text/plain; version=0.0.4; charset=utf-8"
)

type Config struct {
	GitHubUser         string
	GitHubOrganization string
}

func main() {
	handler, err := newServerHandler()
	exitIfError(err)

	if lambdautils.InLambda() {
		lambda.StartHandler(lambdautils.NewLambdaHttpHandlerAdapter(handler))
		return
	}

	logger := logex.StandardLogger()

	exitIfError(runStandaloneServer(
		ossignal.InterruptOrTerminateBackgroundCtx(logger),
		handler,
		logger))
}

func newServerHandler() (http.Handler, error) {
	mux := http.NewServeMux()

	conf, err := getConfig()
	if err != nil {
		return nil, err
	}

	githubClient := github.NewClient(nil)

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		gitHubMetricsReg, err := fetchGitHubMetrics(r.Context(), conf, githubClient)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		expositionOutput := &bytes.Buffer{}

		if err := prompipeclient.GatherToTextExport(gitHubMetricsReg, expositionOutput); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", promContentType)

		fmt.Fprintln(w, expositionOutput.String())
	})

	return mux, nil
}

func fetchGitHubMetrics(
	ctx context.Context,
	conf *Config,
	githubClient *github.Client,
) (*prometheus.Registry, error) {
	gitHubMetrics := promconstmetrics.NewCollector()

	gitHubMetricsReg := prometheus.NewRegistry()
	if err := gitHubMetricsReg.Register(gitHubMetrics); err != nil {
		return nil, err
	}

	if conf.GitHubOrganization != "" {
		page := 0

		for {
			repos, resp, err := githubClient.Repositories.ListByOrg(ctx, conf.GitHubOrganization, &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{Page: page, PerPage: 100},
			})
			if err != nil {
				return nil, err
			}

			timeOfFetch := time.Now()

			for _, repo := range repos {
				pushRepoStats(repo, timeOfFetch, gitHubMetrics, conf.GitHubOrganization)
			}

			if resp.NextPage == 0 {
				break
			}

			page = resp.NextPage
		}
	}

	if conf.GitHubUser != "" {
		page := 0

		for {
			repos, resp, err := githubClient.Repositories.List(ctx, conf.GitHubUser, &github.RepositoryListOptions{
				ListOptions: github.ListOptions{Page: page, PerPage: 100},
			})
			if err != nil {
				return nil, err
			}

			timeOfFetch := time.Now()

			for _, repo := range repos {
				pushRepoStats(repo, timeOfFetch, gitHubMetrics, conf.GitHubUser)
			}

			if resp.NextPage == 0 {
				break
			}

			page = resp.NextPage
		}
	}

	return gitHubMetricsReg, nil
}

func pushRepoStats(
	repo *github.Repository,
	ts time.Time,
	gitHubMetrics *promconstmetrics.Collector,
	owner string,
) {
	push := func(key string, val float64) {
		gitHubMetrics.Observe(gitHubMetrics.Register(key, "", prometheus.Labels{
			"id":    strconv.Itoa(int(*repo.ID)),
			"repo":  *repo.Name,
			"owner": owner,
		}), val, ts)
	}

	push("github_stars", float64(*repo.StargazersCount))
	push("github_watchers", float64(*repo.WatchersCount))
	push("github_size", float64(*repo.Size))
	push("github_forks", float64(*repo.ForksCount))
	push("github_issues_open", float64(*repo.OpenIssuesCount))
}

func runStandaloneServer(ctx context.Context, handler http.Handler, logger *log.Logger) error {
	srv := &http.Server{
		Addr:    ":80",
		Handler: handler,

		ReadHeaderTimeout: 60 * time.Second,
	}

	tasks := taskrunner.New(ctx, logger)

	tasks.Start("listener "+srv.Addr, func(_ context.Context, _ string) error {
		return httputils.RemoveGracefulServerClosedError(srv.ListenAndServe())
	})

	tasks.Start("listenershutdowner", httputils.ServerShutdownTask(srv))

	return tasks.Wait()
}

func getConfig() (*Config, error) {
	cfg := &Config{
		GitHubOrganization: os.Getenv("GITHUB_ORG"),
		GitHubUser:         os.Getenv("GITHUB_USER"),
	}

	if cfg.GitHubOrganization == "" && cfg.GitHubUser == "" {
		return nil, errors.New("GITHUB_ORG and GITHUB_USER both cannot be empty")
	}

	return cfg, nil
}

func exitIfError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
