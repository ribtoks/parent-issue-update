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
	errParentNotFound = errors.New("parent issue not found")
)

type tree struct {
	nodes  map[int]map[int]bool
	issues map[int]*Issue
}

func isParentIssueMark(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	return m == "parent issue" ||
		m == "epic" ||
		m == "parent"
}

func parseIssueNumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
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
		return parseIssueNumber(parts[1])
	}
	return -1, errParentNotFound
}

func NewTree(issues []*github.Issue) *tree {
	t := &tree{}
	for _, i := range issues {
		child := i.GetNumber()
		t.issues[child] = NewIssue(i)

		parent, err := parseParentIssue(i)
		if err != nil {
			log.Printf("Failed to parse parent issue. issue=%v err=%v", i.GetID(), err)
			continue
		}

		t.nodes[parent][child] = true
		log.Printf("Added issues link. parent=%v child=%v", parent, child)
	}

	return t
}

func (t *tree) Issues() []*Issue {
	issues := make([]*Issue, 0, len(t.nodes))
	for p, cm := range t.nodes {
		children := make([]*Issue, 0, len(cm))
		for i, _ := range cm {
			children = append(children, t.issues[i])
		}

		pi := t.issues[p]
		pi.Children = children

		issues = append(issues, pi)
	}

	return issues
}
