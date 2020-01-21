package main

import (
	"context"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/function61/gokit/envvar"
	"github.com/function61/hautomo/pkg/constmetrics"
	"github.com/function61/prompipe/pkg/prompipeclient"
	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"time"
)

type Config struct {
	GitHubOrganization string
	PromPipeEndpoint   string
	PromPipeAuthToken  string
}

func gitHubStatsToPrompipe(ctx context.Context) error {
	conf, err := getConfig()
	if err != nil {
		return err
	}

	client := github.NewClient(nil)

	constMetrics := constmetrics.NewCollector()

	allMetrics := prometheus.NewRegistry()
	if err := allMetrics.Register(constMetrics); err != nil {
		return err
	}

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, conf.GitHubOrganization, opt)
		if err != nil {
			return err
		}

		ts := time.Now()

		for _, repo := range repos {
			pushRepoStats(repo, ts, constMetrics)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	ppc := prompipeclient.New(conf.PromPipeEndpoint, conf.PromPipeAuthToken)

	if err := ppc.Send(ctx, allMetrics); err != nil {
		return err
	}

	/*
		expositionOutput := &bytes.Buffer{}

		if err := prompipeclient.GatherToTextExport(allMetrics, expositionOutput); err != nil {
			return err
		}

		fmt.Println(expositionOutput.String())
	*/

	return nil
}

func pushRepoStats(repo *github.Repository, ts time.Time, metrics *constmetrics.Collector) {
	push := func(key string, val float64) {
		// metrics.Observe(metrics.Register(key, "", "id", strconv.Itoa(int(*repo.ID)), "repo", *repo.Name), val, ts)
		metrics.Observe(metrics.Register(key, "", "repo", *repo.Name), val, ts)
	}

	push("github_stars", float64(*repo.StargazersCount))
	push("github_watchers", float64(*repo.WatchersCount))
	push("github_size", float64(*repo.Size))
	push("github_forks", float64(*repo.ForksCount))
	push("github_issues_open", float64(*repo.OpenIssuesCount))
}

// this handler is driven by Cloudwatch scheduled event
func lambdaHandler(ctx context.Context, req events.CloudWatchEvent) error {
	return gitHubStatsToPrompipe(ctx)
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "dev" {
		if err := gitHubStatsToPrompipe(context.Background()); err != nil {
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

	return &Config{
		GitHubOrganization: getRequiredEnv("GITHUB_ORG"),
		PromPipeEndpoint:   getRequiredEnv("PROMPIPE_ENDPOINT"),
		PromPipeAuthToken:  getRequiredEnv("PROMPIPE_AUTHTOKEN"),
	}, validationError
}
