# Update parent issues using GitHub Action

[![Build](https://github.com/ribtoks/parent-issue-action/workflows/Build%20and%20Test/badge.svg)](https://github.com/ribtoks/parent-issue-action/actions)
[![Integration Test](https://github.com/ribtoks/parent-issue-action/workflows/Integration%20Test/badge.svg)](https://github.com/ribtoks/parent-issue-action/actions)

GitHub Action that updates parent issues that are linked by child issues. When child issue is closed or opened, referenced parent is updated. You can use this to create Epics using GitHub Issues or simply for better tracking of dependent issues. Issue hierarchies are supported too.

In order to link child issues to parent issues, add a line `Parent: #1234` or `Epic: #1234` or `Parent issue: #1234` anywhere in the child issue body.

## Screenshot

![Update result](screenshot.png "Example of created issue")

## Usage

Create a workflow file in your `.github/workflows/` directory with the following contents:

### Basic example

```yaml
name: Epics workflow
on:
  schedule:
    - cron:  '0 5 * * *'
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@master
    - name: Update parent issues
      uses: ribtoks/parent-issue-update@master
      with:
        TOKEN: ${{ secrets.GITHUB_TOKEN }}
        REPO: ${{ github.repository }}
```

> **NOTE:** Please note that currently GitHub has 5000 requests per hour limit so if you are running it on a fresh repository and you have really many issues, you may hit this limit.

This action was designed to run on schedule instead of per issue update event in order to reduce amount of possible race conditions when multiple issues are updated at once and multiple worflows are started in parallel. Currently GitHub Actions do not support cancelling parallel jobs. When this will be supported, it will be safe to use this action per `issue` `opened`/`reopened`/`closed` trigger.

You can use this action together with [TODO issue generator](https://github.com/ribtoks/tdg-github-action) in order to link all TODO isues to the parent issue for better tracking (if you use a specific syntax in the TODO body - see [docs](https://github.com/ribtoks/tdg-github-action#todo-comments)).

### Inputs

| Input                                             | Description                                        |
|------------------------------------------------------|-----------------------------------------------|
| `TOKEN`  | Github token used to create or close issues (required)  |
| `REPO`  | Repository name in the format of `owner/repo` (required)   |
| `DRY_RUN`  | Do not update real real issues (used for debugging) |
| `SYNC_DAYS` | Update parent issues for issue changes in the last `SYNC_DAYS` (defaults to `1`) |
| `MAX_LEVELS` | Keep this deep hierarchy in parent issues (defaults to `0` - unlimited)
| `ADD_CHANGELOG`  | Add a comment with the update changelog to parent issue (default `1` - enabled) |
| `UPDATE_CLOSED`  | Update closed parent issues too (default `0` - disabled) |

Flag values like `DRY_RUN` or `ADD_CHANGELOG` use values `1`/`true`/`y` as ON switch.

If you want to run this action every week, you need to update `SYNC_DAYS` to `7` and update cron job schedule in Action syntax to be `0 0 0 * *` (use [crontab guru](https://crontab.guru/) for help).

If you want to sync all issues at every run, use `all` as a value for `SYNC_DAYS`. This may be useful on the initial integration in the repository.

### Outputs

| Output                                             | Description                                        |
|------------------------------------------------------|-----------------------------------------------|
| `updatedIssues`  | Equals to `1` if updated any issues    |
