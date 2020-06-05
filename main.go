package main

import (
	"context"
	"fmt"
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
	token        string
	owner        string
	repo         string
	syncDays     int
	maxLevels    int
	addChangelog bool
	dryRun       bool
	updateClosed bool
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
		owner:        r[0],
		repo:         r[1],
		token:        os.Getenv("INPUT_TOKEN"),
		dryRun:       flagToBool(os.Getenv("INPUT_DRY_RUN")),
		addChangelog: flagToBool(os.Getenv("INPUT_ADD_CHANGELOG")),
		updateClosed: flagToBool(os.Getenv("INPUT_UPDATE_CLOSED")),
	}

	var err error

	syncDays := os.Getenv("INPUT_SYNC_DAYS")
	e.syncDays, err = strconv.Atoi(syncDays)
	if err != nil {
		if strings.ToLower(syncDays) == "all" {
			e.syncDays = -1
		} else {
			e.syncDays = defaultSyncDays
		}
	}

	e.maxLevels, err = strconv.Atoi(os.Getenv("INPUT_MAX_LEVELS"))
	if err != nil {
		e.maxLevels = defaultMaxLevels
	}

	return e
}

func (e *env) debugPrint() {
	log.Printf("Repo: %v", e.repo)
	log.Printf("Owner: %v", e.owner)
	log.Printf("Sync days: %v", e.syncDays)
	log.Printf("Max levels: %v", e.maxLevels)
	log.Printf("Dry run: %v", e.dryRun)
	log.Printf("Add comments: %v", e.addChangelog)
	log.Printf("Update closed: %v", e.updateClosed)
}

func (s *service) fetchGithubIssues() ([]*github.Issue, error) {
	var allIssues []*github.Issue

	opt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: defaultIssuesPerPage},
	}

	if s.env.syncDays > 0 {
		opt.Since = time.Now().AddDate(0 /*year*/, 0 /*month*/, -s.env.syncDays)
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
	log.Printf("Fetched github issues. count=%v", len(allIssues))

	return allIssues, nil
}

func (s *service) fetchIssuesByID(issues []int) ([]*github.Issue, error) {
	log.Printf("Fetching issues by ID. count=%v", len(issues))
	var wg sync.WaitGroup

	var allIssues []*github.Issue
	for _, i := range issues {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			issue, _, err := s.client.Issues.Get(s.ctx, s.env.owner, s.env.repo, id)
			if err != nil {
				log.Printf("Failed to retrieve an issue. issue=%v err=%v", id, err)
				return
			}

			allIssues = append(allIssues, issue)
		}(i)
	}

	log.Printf("Waiting for issues to be fetched by ID...")
	wg.Wait()

	return allIssues, nil
}

func createComment(changelog []string) string {
	if len(changelog) == 0 {
		return ""
	}

	var str strings.Builder
	str.WriteString("Issue update changelog:\n")
	for _, s := range changelog {
		str.WriteString(fmt.Sprintf("- %s\n", s))
	}
	return str.String()
}

func (s *service) updateIssue(i *Issue, body string, changelog []string) {
	defer s.wg.Done()

	log.Printf("About to update an issue. issue=%v", i.ID)
	if s.env.dryRun {
		log.Printf("Dry run mode.")
		return
	}

	req := &github.IssueRequest{
		Body: &body,
	}
	_, _, err := s.client.Issues.Edit(s.ctx, s.env.owner, s.env.repo, i.ID, req)

	if err != nil {
		log.Printf("Error while editing an issue. issue=%v err=%v", i.ID, err)
		return
	}

	log.Printf("Updated an issue. issue=%v", i.ID)

	if s.env.addChangelog && len(changelog) > 0 {
		body := createComment(changelog)
		comment := &github.IssueComment{
			Body: &body,
		}
		_, _, err = s.client.Issues.CreateComment(s.ctx, s.env.owner, s.env.repo, i.ID, comment)
		if err != nil {
			log.Printf("Error while adding a comment. issue=%v err=%v", i.ID, err)
			return
		}

		log.Printf("Added a comment to the issue. issue=%v", i.ID)
	}
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
		fmt.Println(fmt.Sprintf(`::set-output name=updatedIssues::%s`, "1"))
		return
	}

	tr := NewTree(ghIssues)
	missing, err := svc.fetchIssuesByID(tr.missing)
	if err != nil {
		log.Panic(err)
	}
	tr.AddParentIssues(missing)
	issues := tr.Issues()

	e := &Editor{
		MaxLevels: svc.env.maxLevels,
	}

	for _, i := range issues {
		canProcess := i.IsOpened() || (i.IsClosed() && svc.env.updateClosed)
		if !canProcess {
			log.Printf("Skipping issue update. issue=%v status=%v", i.ID, i.Status)
			continue
		}

		body, changeLog, err := e.Update(i, true /*add missing*/)
		if err != nil {
			log.Printf("Failed to update issue body. issue=%v err=%v", i.ID, err)
			continue
		}

		if body == i.Body {
			log.Printf("Skipping identical issue body. issue=%v", i.ID)
			continue
		}

		svc.wg.Add(1)
		go svc.updateIssue(i, body, changeLog)
	}

	log.Printf("Waiting for issue update to finish...")
	svc.wg.Wait()

	fmt.Println(fmt.Sprintf(`::set-output name=updatedIssues::%s`, "1"))

	// help logger to flush
	time.Sleep(1 * time.Second)
}
