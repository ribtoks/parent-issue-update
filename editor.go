package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"unicode"
)

const (
	issueSectionHead = "### Child issues:"
	eol              = "\n"
	nextLevel        = 1
	spacesPerLevel   = 2
)

var (
	errAlreadyAdded = errors.New("issue was already added")
	errLevelTooDeep = errors.New("level is too deep")
)

type stack struct {
	data []*Issue
}

func (s *stack) push(i *Issue) {
	s.data = append(s.data, i)
}

func (s *stack) empty() bool {
	return len(s.data) == 0
}

func (s *stack) top() *Issue {
	l := len(s.data)
	if l == 0 {
		return nil
	}

	return s.data[l-1]
}

func (s *stack) pop() {
	l := len(s.data)
	if l > 0 {
		s.data = s.data[:l-1]
	}
}

type editContext struct {
	ChangeLog  []string
	AddMissing bool
	Processed  map[int]bool
	Stack      *stack
}

func isKnownError(err error) bool {
	return err == errAlreadyAdded ||
		err == errLevelTooDeep
}

type Editor struct {
	MaxLevels int
}

func (e *Editor) formatForEmpty(i *Issue, level int, str io.StringWriter, skipMap map[int]bool) error {
	if e.MaxLevels > 0 && level >= e.MaxLevels {
		return errLevelTooDeep
	}

	if _, ok := skipMap[i.ID]; ok {
		log.Printf("Skipping processed issue. issue=%v", i.ID)
		return errAlreadyAdded
	}

	if _, err := str.WriteString(i.FormatTitle(level*spacesPerLevel) + eol); err != nil {
		return err
	}

	if len(i.Children) == 0 {
		return nil
	}

	for _, ci := range i.Children {
		if err := e.formatForEmpty(ci, level+nextLevel, str, skipMap); err != nil {
			if !isKnownError(err) {
				return err
			}
		}
	}

	return nil
}

func (c *editContext) log(s string) {
	log.Printf("Log update. change=%v", s)
	c.ChangeLog = append(c.ChangeLog, s)
}

func (c *editContext) logUpdate(i *Issue) {
	status := "opened"
	if i.Status == StatusClosed {
		status = "closed"
	}
	c.log(fmt.Sprintf("Updated child issue #%v. New status: %v", i.ID, status))
}

func (e *Editor) appendNewSection(i *Issue, ctx *editContext) (string, error) {
	var str strings.Builder

	str.WriteString(fmt.Sprintf("%s\n\n", issueSectionHead))
	skipMap := make(map[int]bool)

	for _, ci := range i.Children {
		if err := e.formatForEmpty(ci, 0 /*level*/, &str, skipMap); err != nil {
			return "", err
		}
	}

	result := ""
	if len(i.Body) > 0 {
		result = strings.TrimRight(i.Body, " \n\t") + "\n\n" + str.String()
	} else {
		result = str.String()
	}

	ctx.log(fmt.Sprintf("Appended new block with %v child issue(s)", len(i.Children)))

	return result, nil
}

func countPrefixSpaces(s string) int {
	count := 0

	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			break
		}
		count++
	}

	return count
}

func parseIssueID(s string) (int, error) {
	hashStart := strings.LastIndex(s, "#")
	issueStr := strings.TrimSpace(s[hashStart+1:])

	return strconv.Atoi(issueStr)
}

func isAllWhitespace(s string) bool {
	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			return false
		}
	}

	return true
}

func (e *Editor) addMissing(parent *Issue, str io.StringWriter, ctx *editContext) {
	if parent == nil {
		return
	}

	log.Printf("Adding missing issues. parent=%v level=%v", parent.ID, parent.Level)
	added := 0

	for _, ci := range parent.Children {
		if err := e.formatForEmpty(ci, parent.Level+1, str, ctx.Processed); err != nil {
			if err != errAlreadyAdded {
				log.Printf("Error while appending new child issues. err=%v", err)
			}
		} else {
			added++
		}
	}

	if added > 0 {
		ctx.log(fmt.Sprintf("Appended %v new child issue(s) on level %v", added, parent.Level+1))
	}
}

func (e *Editor) updateIssues(i *Issue, start int, ctx *editContext) string {
	var str strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(i.Body[start:]))
	issueMap := i.ToMap()
	i.Level = -1
	ctx.Stack.push(i)

	for scanner.Scan() {
		line := scanner.Text()
		if isAllWhitespace(line) {
			str.WriteString(eol)
			continue
		}

		spaces := countPrefixSpaces(line)
		log.Printf("Processing child issue. line=%v spaces=%v", line, spaces)

		if e.MaxLevels > 0 && spaces/2 >= e.MaxLevels {
			log.Printf("Issue is above max level. level=%v", e.MaxLevels)
			str.WriteString(line + eol)
			continue
		}

		id, err := parseIssueID(line)
		if err != nil {
			log.Printf("Failed to parse issue ID. line=%v err=%v", line, err)
			str.WriteString(line + eol)

			continue
		}

		if ctx.Stack.top().Level >= spaces/2 && ctx.AddMissing {
			e.addMissing(ctx.Stack.top(), &str, ctx)
			ctx.Stack.pop()
		}

		ci, ok := issueMap[id]
		if !ok {
			log.Printf("Failed to find child issue by ID. id=%v", id)
			str.WriteString(line + eol)

			continue
		}

		log.Printf("Found child issue. id=%v status=%v spaces=%v", ci.ID, ci.Status, ci.Level)
		ci.Level = spaces / 2
		ctx.Stack.push(ci)
		ctx.Processed[id] = true
		title := ci.FormatTitle(spaces)
		if title != line {
			ctx.logUpdate(ci)
		}
		str.WriteString(title + eol)
	}

	for ctx.AddMissing && !ctx.Stack.empty() {
		e.addMissing(ctx.Stack.top(), &str, ctx)
		ctx.Stack.pop()
	}

	return i.Body[:start] + str.String()
}

func (e *Editor) Update(i *Issue, addMissing bool) (string, []string, error) {
	if i == nil {
		return "", nil, nil
	}

	if len(i.Children) == 0 {
		return i.Body, nil, nil
	}

	ctx := &editContext{
		ChangeLog:  make([]string, 0),
		AddMissing: addMissing,
		Processed:  make(map[int]bool),
		Stack:      &stack{data: make([]*Issue, 0)},
	}

	if len(i.Body) == 0 {
		body, err := e.appendNewSection(i, ctx)
		return body, ctx.ChangeLog, err
	}

	sectionStart := strings.LastIndex(i.Body, issueSectionHead)
	if sectionStart == -1 {
		body, err := e.appendNewSection(i, ctx)
		return body, ctx.ChangeLog, err
	}

	body := e.updateIssues(i, sectionStart+len(issueSectionHead), ctx)
	return body, ctx.ChangeLog, nil
}
