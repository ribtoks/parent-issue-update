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
	if s[0] != '#' {
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
		return parseIssueNumber(parts[1])
	}
	return -1, errParentNotFound
}

func NewTree(issues []*github.Issue) *tree {
	t := &tree{
		nodes:  make(map[int]map[int]bool),
		issues: make(map[int]*Issue),
	}
	for _, i := range issues {
		child := i.GetNumber()
		t.issues[child] = NewIssue(i)

		parent, err := parseParentIssue(i)
		if err != nil {
			log.Printf("Failed to parse parent issue. issue=%v err=%v", i.GetID(), err)
			continue
		}

		t.addNode(parent, child)
		log.Printf("Added issues link. parent=%v child=%v", parent, child)
	}

	return t
}

func (t *tree) addNode(parent, child int) {
	if _, ok := t.nodes[parent]; !ok {
		t.nodes[parent] = make(map[int]bool)
	}

	t.nodes[parent][child] = true
}

func (t *tree) Issues() []*Issue {
	log.Printf("Making list out of issue tree")
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

		pi := t.issues[p]
		pi.Children = children

		issues = append(issues, pi)
	}

	return issues
}
