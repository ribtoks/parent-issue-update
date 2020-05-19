package main

import (
	"bufio"
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

type Editor struct {
	MaxLevels int
}

func (e *Editor) formatForEmpty(i *Issue, level int, str io.StringWriter) error {
	if e.MaxLevels > 0 && level >= e.MaxLevels {
		return nil
	}

	if _, err := str.WriteString(i.FormatTitle(level*spacesPerLevel) + eol); err != nil {
		return err
	}

	if len(i.Children) == 0 {
		return nil
	}

	for _, ci := range i.Children {
		if err := e.formatForEmpty(ci, level+nextLevel, str); err != nil {
			return err
		}
	}

	return nil
}

func (e *Editor) appendNewSection(i *Issue) error {
	var str strings.Builder

	str.WriteString(fmt.Sprintf("%s\n\n", issueSectionHead))

	for _, ci := range i.Children {
		if err := e.formatForEmpty(ci, 0 /*level*/, &str); err != nil {
			return err
		}
	}

	if len(i.Body) > 0 {
		i.Body = strings.TrimRight(i.Body, " \n\t") + "\n\n" + str.String()
	} else {
		i.Body = str.String()
	}

	return nil
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

func (e *Editor) updateIssues(i *Issue, start int) {
	var str strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(i.Body[start:]))
	issueMap := i.ToMap()

	for scanner.Scan() {
		line := scanner.Text()
		if isAllWhitespace(line) {
			str.WriteString(eol)
			continue
		}

		spaces := countPrefixSpaces(line)
		if e.MaxLevels > 0 && spaces/2 >= e.MaxLevels {
			log.Printf("Issue is below max level. level=%v", e.MaxLevels)
			str.WriteString(line + eol)
			continue
		}

		log.Printf("Processing child issue. line=%v spaces=%v", line, spaces)
		id, err := parseIssueID(line)

		if err != nil {
			log.Printf("Failed to parse issue ID. line=%v err=%v", line, err)
			str.WriteString(line + eol)

			continue
		}

		ci, ok := issueMap[id]
		if !ok {
			log.Printf("Failed to find child issue by ID. id=%v", id)
			str.WriteString(line + eol)

			continue
		}

		log.Printf("Found child issue. id=%v status=%v", ci.ID, ci.Status)
		str.WriteString(ci.FormatTitle(spaces) + eol)
	}

	i.Body = i.Body[:start] + str.String()
}

func (e *Editor) Update(i *Issue) error {
	if i == nil || len(i.Children) == 0 {
		return nil
	}

	log.Printf("Editing issue body. issue=%v children=%v", i.ID, len(i.Children))

	if len(i.Body) == 0 {
		return e.appendNewSection(i)
	}

	sectionStart := strings.LastIndex(i.Body, issueSectionHead)
	if sectionStart == -1 {
		return e.appendNewSection(i)
	}

	e.updateIssues(i, sectionStart+len(issueSectionHead))

	return nil
}
