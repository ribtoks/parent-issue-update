package main

import (
	"bufio"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/google/go-github/v31/github"
)

var (
	errParentNotFound   = errors.New("parent issue not found")
	errWrongIssueSyntax = errors.New("wrong issue syntax")
)

type tree struct {
	// map from parent to child issue
	nodes   map[int]map[int]bool
	issues  map[int]*Issue
	missing []int
}

func isParentIssueMark(m string) bool {
	if len(m) == 0 {
		return false
	}

	m = strings.ToLower(strings.TrimSpace(m))

	return m == "parent issue" ||
		m == "epic" ||
		m == "parent"
}

func parseIssueNumber(s string) (int, error) {
	s = strings.TrimSpace(s)

	// 10 digits of max-int + '#'
	if s[0] != '#' || len(s) > 11 {
		return -1, errWrongIssueSyntax
	}

	return strconv.Atoi(s[1:])
}

func parseParentIssue(i *github.Issue) (int, error) {
	scanner := bufio.NewScanner(strings.NewReader(i.GetBody()))
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.Contains(line, "#") {
			continue
		}

		if !strings.Contains(line, ":") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		if !isParentIssueMark(parts[0]) {
			continue
		}

		issue, err := parseIssueNumber(parts[1])
		if err != nil {
			log.Printf("Failed to parse parent issue. line=%v err=%v", line, err)
			continue
		}

		return issue, nil
	}

	return -1, errParentNotFound
}

func NewTree(issues []*github.Issue) *tree {
	t := &tree{
		nodes:   make(map[int]map[int]bool),
		issues:  make(map[int]*Issue),
		missing: make([]int, 0),
	}

	for _, i := range issues {
		child := i.GetNumber()
		t.issues[child] = NewIssue(i)

		parent, err := parseParentIssue(i)
		if err != nil {
			log.Printf("Failed to parse parent issue. issue=%v err=%v", i.GetNumber(), err)
			continue
		}

		t.addNode(parent, child)
	}

	for p, _ := range t.nodes {
		if _, ok := t.issues[p]; !ok {
			t.missing = append(t.missing, p)
		}
	}

	log.Printf("Processed missing parent issues. count=%v", len(t.missing))

	return t
}

func (t *tree) addNode(parent, child int) {
	if _, ok := t.nodes[parent]; !ok {
		t.nodes[parent] = make(map[int]bool)
	}

	t.nodes[parent][child] = true
	log.Printf("Added issues link. parent=%v child=%v", parent, child)
}

func (t *tree) AddParentIssues(issues []*github.Issue) {
	log.Printf("Adding additional parent issues. count=%v", len(issues))
	for _, i := range issues {
		issue := NewIssue(i)

		if _, ok := t.issues[issue.ID]; ok {
			log.Printf("Parent issue is already added. issue=%v", issue.ID)
			continue
		}

		t.issues[issue.ID] = issue
	}
}

func (t *tree) Issues() []*Issue {
	log.Printf("Making a list out of issue tree. nodes_count=%v", len(t.nodes))
	issues := make([]*Issue, 0, len(t.nodes))

	for p, cm := range t.nodes {
		log.Printf("Generating children list. parent=%v children_count=%v", p, len(cm))
		children := make([]*Issue, 0, len(cm))

		for i, _ := range cm {
			if _, ok := t.issues[i]; !ok {
				log.Printf("Child issue is not found. issue=%v", i)
				continue
			}

			children = append(children, t.issues[i])
		}

		pi, ok := t.issues[p]
		if !ok {
			log.Printf("Failed to find an issue. issue=%v", p)
			continue
		}

		pi.Children = children

		issues = append(issues, pi)
	}

	log.Printf("Generated list of parent issues. count=%v", len(issues))

	return issues
}
