package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/envvar"
	"github.com/function61/gokit/promconstmetrics"
	"github.com/function61/prompipe/pkg/prompipeclient"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"strconv"
	"time"
)

type Config struct {
	GitHubUser         string
	GitHubOrganization string
	PromPipeEndpoint   string
	PromPipeAuthToken  string
}

func pushToPromPipe(
	ctx context.Context,
	allMetrics *prometheus.Registry,
	conf Config,
) error {
	return prompipeclient.New(conf.PromPipeEndpoint, conf.PromPipeAuthToken).Send(ctx, allMetrics)
}

func printResults(
	ctx context.Context,
	allMetrics *prometheus.Registry,
	conf Config,
) error {
	expositionOutput := &bytes.Buffer{}

	if err := prompipeclient.GatherToTextExport(allMetrics, expositionOutput); err != nil {
		return err
	}

	fmt.Println(expositionOutput.String())

	return nil
}

func fetchGitHubStats(
	ctx context.Context,
	resultHandler func(context.Context, *prometheus.Registry, Config) error,
) error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	githubClient := github.NewClient(nil)

	gitHubMetrics := promconstmetrics.NewCollector()

	allMetrics := prometheus.NewRegistry()
	if err := allMetrics.Register(gitHubMetrics); err != nil {
		return err
	}

	if conf.GitHubOrganization != "" {
		page := 0

		for {
			repos, resp, err := githubClient.Repositories.ListByOrg(ctx, conf.GitHubOrganization, &github.RepositoryListByOrgOptions{
				ListOptions: github.ListOptions{Page: page, PerPage: 100},
			})
			if err != nil {
				return err
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
				return err
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

	return resultHandler(ctx, allMetrics, *conf)
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

// this handler is driven by Cloudwatch scheduled event
func lambdaHandler(ctx context.Context, req events.CloudWatchEvent) error {
	return fetchGitHubStats(ctx, pushToPromPipe)
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "dev" {
		if err := fetchGitHubStats(context.Background(), printResults); err != nil {
			panic(err)
		}
		return
	}

	lambda.Start(lambdaHandler)
}

func getConfig() (*Config, error) {
	var validationError error
	getRequiredEnv := func(key string) string {
		val, err := envvar.Get(key)
		if err != nil {
			validationError = err
		}

		return val
	}

	cfg := &Config{
		GitHubOrganization: os.Getenv("GITHUB_ORG"),
		GitHubUser:         os.Getenv("GITHUB_USER"),
		PromPipeEndpoint:   getRequiredEnv("PROMPIPE_ENDPOINT"),
		PromPipeAuthToken:  getRequiredEnv("PROMPIPE_AUTHTOKEN"),
	}

	if cfg.GitHubOrganization == "" && cfg.GitHubUser == "" {
		return nil, errors.New("GITHUB_ORG and GITHUB_USER both cannot be empty")
	}

	return cfg, validationError
}
