package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
)

const (
	defaultIssuesPerPage = 200
	defaultSyncDays      = 1
	defaultMaxLevels     = 0
)

type env struct {
	token     string
	owner     string
	repo      string
	syncDays  int
	maxLevels int
	dryRun    bool
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
	r := strings.Split(os.Getenv("INPUT_REPO"), "/")

	e := &env{
		owner:  r[0],
		repo:   r[1],
		token:  os.Getenv("INPUT_TOKEN"),
		dryRun: flagToBool(os.Getenv("INPUT_DRY_RUN")),
	}

	var err error

	e.syncDays, err = strconv.Atoi(os.Getenv("INPUT_SYNC_DAYS"))
	if err != nil {
		e.syncDays = defaultSyncDays
	}

	e.maxLevels, err = strconv.Atoi(os.Getenv("INPUT_MAX_LEVELS"))
	if err != nil {
		e.maxLevels = defaultMaxLevels
	}

	return e
}

func (e *env) debugPrint() {
	log.Printf("Sync days: %v", e.syncDays)
	log.Printf("Max levels: %v", e.maxLevels)
	log.Printf("Dry run: %v", e.dryRun)
}

func (s *service) fetchGithubIssues() ([]*github.Issue, error) {
	var allIssues []*github.Issue

	opt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: defaultIssuesPerPage},
		Since:       time.Now().AddDate(0 /*year*/, 0 /*month*/, -s.env.syncDays),
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
	log.Printf("Fetched github todo issues. count=%v", len(allIssues))

	return allIssues, nil
}

func (s *service) updateIssue(i *Issue) {
	defer s.wg.Done()

	log.Printf("About to update an issue. issue=%v", i.ID)
	if s.env.dryRun {
		log.Printf("Dry run mode.")
		return
	}

	req := &github.IssueRequest{
		Body: &i.Body,
	}
	_, _, err := s.client.Issues.Edit(s.ctx, s.env.owner, s.env.repo, i.ID, req)

	if err != nil {
		log.Printf("Error while editing an issue. issue=%v err=%v", i.ID, err)
	}

	log.Printf("Updated issue. issue=%v", i.ID)
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

	ghIssues, err := svc.fetchGithubIssues()
	if err != nil {
		log.Panic(err)
	}

	if len(ghIssues) == 0 {
		return
	}

	tr := NewTree(ghIssues)
	issues := tr.Issues()

	e := &Editor{
		MaxLevels: svc.env.maxLevels,
	}

	for _, i := range issues {
		oldBody := i.Body
		err := e.Update(i)
		if err != nil {
			log.Printf("Failed to update issue body. issue=%v err=%v", i.ID, err)
			continue
		}

		if oldBody != i.Body {
			log.Printf("Skipping identical issue body. issue=%v", i.ID)
			continue
		}

		svc.wg.Add(1)
		go svc.updateIssue(i)
	}

	log.Printf("Waiting for issue update to finish")
	svc.wg.Wait()
}
