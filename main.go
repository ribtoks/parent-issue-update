package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
)

const (
	defaultIssuesPerPage = 200
)

type env struct {
	token  string
	dryRun bool
}

type service struct {
	ctx    context.Context
	client *github.Client
	env    *env
	wg     sync.WaitGroup
}

func flagToBool(s string) bool {
	s = strings.ToLower(s)
	return s == "1" || s == "true" || s == "y" || s == "yes"
}

func environment() *env {
	e := &env{
		token:  os.Getenv("INPUT_TOKEN"),
		dryRun: flagToBool(os.Getenv("INPUT_DRY_RUN")),
	}

	return e
}

func (e *env) debugPrint() {
	log.Printf("Dry run: %v", e.dryRun)
}

func (s *service) fetchGithubIssues() ([]*github.Issue, error) {
	var allIssues []*github.Issue

	opt := &github.IssueListByRepoOptions{
		Labels:      []string{s.env.label},
		State:       "all",
		ListOptions: github.ListOptions{PerPage: defaultIssuesPerPage},
	}

	for {
		issues, resp, err := s.client.Issues.ListByRepo(s.ctx, s.env.owner, s.env.repo, opt)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}

		opt.Page = resp.NextPage
	}
	log.Printf("Fetched github todo issues. count=%v label=%v", len(allIssues), s.env.label)

	return allIssues, nil
}

func main() {
	log.SetOutput(os.Stdout)
	env := environment()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: env.token},
	)
	tc := oauth2.NewClient(ctx, ts)

	svc := &service{
		ctx:    ctx,
		client: github.NewClient(tc),
		env:    env,
	}

	env.debugPrint()

	issues, err := svc.fetchGithubIssues()
	if err != nil {
		log.Panic(err)
	}
}
