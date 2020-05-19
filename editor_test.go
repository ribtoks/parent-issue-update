package main

import (
	"fmt"
	"log"
	"testing"
)

func createIssuesImpl(children, level, recurse, id int, status IssueStatus) *Issue {
	issue := &Issue{
		ID:     id,
		Title:  fmt.Sprintf("Child Issue id(%v) level(%v)", id, level),
		Status: status,
	}

	if children > 0 {
		issue.Children = make([]*Issue, children)
	}

	for i := 0; i < children; i++ {
		// yes, for tests we support not more than 10 subissues
		issueID := 10*id + i

		if recurse > 0 {
			issue.Children[i] = createIssuesImpl(children, level+1, recurse-1, issueID, status)
		} else {
			issue.Children[i] = &Issue{
				Title: fmt.Sprintf("Child Issue id(%v) level(%v)", issueID, level+1),
			}
		}

		issue.Children[i].ID = issueID
		issue.Children[i].Status = status

		log.Printf("Created issue: %v", issue.Children[i].FormatTitle(0))
	}
	return issue
}

func createIssues(children, level, recurse int, status IssueStatus) *Issue {
	return createIssuesImpl(children, level, recurse, 1 /*parent id*/, status)
}

func EditorSuite(t *testing.T, e *Editor, i *Issue, initialBody, expectedBody string) {
	i.Body = initialBody
	err := e.Update(i)
	if err != nil {
		t.Fatal(err)
	}

	if i.Body != expectedBody {
		t.Errorf("Body does not match. actual=%v expected=%v", i.Body, expectedBody)
	}
}

func EditSuite(t *testing.T, i *Issue, initialBody, expectedBody string) {
	e := &Editor{}
	EditorSuite(t, e, i, initialBody, expectedBody)
}

func TestNoChildren(t *testing.T) {
	body := `
	abcd

	efgh
	`
	issue := createIssues(
		0 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, body)
}

func TestAddOneChildSingleLineBody(t *testing.T) {
	body := "abcd"
	expected := `abcd

### Child issues:

- [ ] Child Issue id(10) level(1) #10
`
	issue := createIssues(

		1 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestAddOneChildEmptyBody(t *testing.T) {
	body := ""
	expected := `### Child issues:

- [ ] Child Issue id(10) level(1) #10
`
	issue := createIssues(
		1 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestAddFewChildrenSingleLineBody(t *testing.T) {
	body := "abcd"
	expected := `abcd

### Child issues:

- [ ] Child Issue id(10) level(1) #10
- [ ] Child Issue id(11) level(1) #11
`
	issue := createIssues(
		2 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestAddHierarchySingleLineBody(t *testing.T) {
	body := "abcd"
	expected := `abcd

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [ ] Child Issue id(100) level(2) #100
  - [ ] Child Issue id(101) level(2) #101
- [ ] Child Issue id(11) level(1) #11
  - [ ] Child Issue id(110) level(2) #110
  - [ ] Child Issue id(111) level(2) #111
`
	issue := createIssues(
		2 /*children*/, 0 /*level*/, 1 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestAddHierarchyMaxLevelsBody(t *testing.T) {
	body := "abcd"
	expected := `abcd

### Child issues:

- [ ] Child Issue id(10) level(1) #10
- [ ] Child Issue id(11) level(1) #11
`
	issue := createIssues(
		2 /*children*/, 0 /*level*/, 1 /*recurse*/, StatusOpened)
	EditorSuite(t,
		&Editor{MaxLevels: 1},
		issue, body, expected)
}

func TestAddFewChildrenNewlines(t *testing.T) {
	body := `abcd  

`
	expected := `abcd

### Child issues:

- [ ] Child Issue id(10) level(1) #10
- [ ] Child Issue id(11) level(1) #11
`
	issue := createIssues(
		2 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestFewChildrenEmptyBody(t *testing.T) {
	body := ""
	expected := `### Child issues:

- [ ] Child Issue id(10) level(1) #10
- [ ] Child Issue id(11) level(1) #11
- [ ] Child Issue id(12) level(1) #12
`
	issue := createIssues(
		3 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestUpdateCheckOneChildEmptyBody(t *testing.T) {
	body := `### Child issues:

- [ ] Child Issue id(10) level(1) #10
`

	expected := `### Child issues:

- [x] Child Issue id(10) level(1) #10
`

	issue := createIssues(
		1 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusClosed)
	EditSuite(t, issue, body, expected)
}

func TestUpdateUncheckOneChildEmptyBody(t *testing.T) {
	body := `### Child issues:

- [x] Child Issue id(10) level(1) #10
`

	expected := `### Child issues:

- [ ] Child Issue id(10) level(1) #10
`

	issue := createIssues(
		1 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestUpdateCheckFewChildrenNonEmptyBody(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [ ] Child Issue id(11) level(1) #11
- [ ] Child Issue id(10) level(1) #10
`

	expected := `abcd
efgh

### Child issues:

- [x] Child Issue id(11) level(1) #11
- [x] Child Issue id(10) level(1) #10
`

	issue := createIssues(
		2 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusClosed)
	EditSuite(t, issue, body, expected)
}

func TestUpdateUncheckFewChildrenNonEmptyBody(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [x] Child Issue id(11) level(1) #11
- [x] Child Issue id(10) level(1) #10
`

	expected := `abcd
efgh

### Child issues:

- [ ] Child Issue id(11) level(1) #11
- [ ] Child Issue id(10) level(1) #10
`

	issue := createIssues(
		2 /*children*/, 0 /*level*/, 0 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestMultiLevelHangingChildrenUpdate(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [x] Child Issue id(10) level(1) #10
  - [x] Child Issue id(100) level(2) #100
    - [x] Child Issue id(1000) level(3) #1000
      - [x] Child Issue id(10000) level(4) #10000
`

	expected := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [ ] Child Issue id(100) level(2) #100
    - [ ] Child Issue id(1000) level(3) #1000
      - [ ] Child Issue id(10000) level(4) #10000
`

	issue := createIssues(
		1 /*children*/, 0 /*level*/, 3 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestMultiLevelOneChildUpdate(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [x] Child Issue id(10) level(1) #10
  - [x] Child Issue id(100) level(2) #100
- [x] Child Issue id(11) level(1) #11
`

	expected := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [ ] Child Issue id(100) level(2) #100
- [ ] Child Issue id(11) level(1) #11
`

	issue := createIssues(
		2 /*children*/, 0 /*level*/, 1 /*recurse*/, StatusOpened)
	EditSuite(t, issue, body, expected)
}

func TestMultiLevelFewChildrenUpdate(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [ ] Child Issue id(100) level(2) #100
  - [x] Child Issue id(101) level(2) #101
    - [x] Child Issue id(1010) level(3) #1010
    - [ ] Child Issue id(1011) level(3) #1011
- [x] Child Issue id(11) level(1) #11
  - [ ] Child Issue id(110) level(2) #110
`

	expected := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [x] Child Issue id(100) level(2) #100
  - [ ] Child Issue id(101) level(2) #101
    - [x] Child Issue id(1010) level(3) #1010
    - [x] Child Issue id(1011) level(3) #1011
- [x] Child Issue id(11) level(1) #11
  - [ ] Child Issue id(110) level(2) #110
`

	issue := createIssues(
		2 /*children*/, 0 /*level*/, 2 /*recurse*/, StatusOpened)
	issue.Children[0].Children[0].Status = StatusClosed
	issue.Children[0].Children[1].Children[0].Status = StatusClosed
	issue.Children[0].Children[1].Children[1].Status = StatusClosed
	issue.Children[1].Status = StatusClosed

	EditSuite(t, issue, body, expected)
}

func TestMultiLevelUpdateMaxLevel(t *testing.T) {
	body := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [ ] Child Issue id(100) level(2) #100
  - [x] Child Issue id(101) level(2) #101
    - [x] Child Issue id(1010) level(3) #1010
    - [ ] Child Issue id(1011) level(3) #1011
- [x] Child Issue id(11) level(1) #11
  - [ ] Child Issue id(110) level(2) #110
`

	expected := `abcd
efgh

### Child issues:

- [ ] Child Issue id(10) level(1) #10
  - [x] Child Issue id(100) level(2) #100
  - [ ] Child Issue id(101) level(2) #101
    - [x] Child Issue id(1010) level(3) #1010
    - [ ] Child Issue id(1011) level(3) #1011
- [x] Child Issue id(11) level(1) #11
  - [ ] Child Issue id(110) level(2) #110
`

	issue := createIssues(
		2 /*children*/, 0 /*level*/, 2 /*recurse*/, StatusOpened)
	issue.Children[0].Children[0].Status = StatusClosed
	issue.Children[0].Children[1].Children[0].Status = StatusClosed
	issue.Children[0].Children[1].Children[1].Status = StatusClosed
	issue.Children[1].Status = StatusClosed

	EditorSuite(t, &Editor{MaxLevels: 2}, issue, body, expected)
}
