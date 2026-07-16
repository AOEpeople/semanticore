# Semanticore Release Bot 🤖 🦁 🐉

## About

Your friendly Semanticore Release bot helps maintaining the changelog for a project and automates the related tagging process.

## How to use it

Semanticore runs along every pipeline in the main branch, and will analyze the commit messages.

It maintains an open Merge Request for the project with all the required Changelog adjustments.

It detects the current version and suggests the next version based on the changes made.

Once a release commit is detected, it will automatically create the related Git tag on the next pipeline run.

## Conventions

* Commit messages should follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) so semanticore can decide whether a minor or patch level release is required.
* Releases are indicated with a commit with a commit messages which should match: `Release vX.Y.Z`

### Supported Commit Types

Currently Semanticore supports the following commit types:

| Type             | Prefixes                   | Meaning                                  |
|------------------|----------------------------|------------------------------------------|
| 🆕 Feature       | `feat`                     | New Feature, creates a minor commit      |
| 🚨 Security Fix  | `sec`                      | Security relevant change/fix             |
| 👾 Bugfix        | `fix`, `bug`               | Bugfix                                   |
| 🛡 Test          | `test`                     | (Unit-)Tests                             |
| 🔁 Refactor      | `refactor`, `rework`       | Refactorings or reworking                |
| 🤖 Devops/CI     | `ops`, `ci`, `cd`, `build` | Operations, Build, CI/CD, Pipelines      |
| 📚 Documentation | `doc`                      | Documentation                            |
| ⚡️ Performance   | `perf`                     | Performance improvements                 |
| 🧹 Chore         | `chore`, `update`          | Chores, (Dependency-)Updates             |
| 📝 Other         | everything else            | Everything not matched by another prefix |

### Major versions

To enable support for major releases (breaking APIs), use the `-major` flag.

## Configuration

The `SEMANTICORE_TOKEN` is required - that's a Gitlab or Github Token which has basic contributor rights and allows to perform the related Git and API operations.

### Sign Key Configuration

To enable GPG signing of commits, you have two options:

- Use `SEMANTICORE_SIGN_KEY` environment variable containing the actual GPG private key
- Use `SEMANTICORE_SIGN_KEY_FILE` environment variable or the command line option `-sign-key-file`
  specifying the path to a file containing the
  GPG private key

If neither is provided, commits will not be signed.

### Set Author and committer

Semanticore respects [Git Environment variables](https://git-scm.com/book/en/v2/Git-Internals-Environment-Variables)

* `GIT_AUTHOR_NAME`
* `GIT_AUTHOR_EMAIL`
* `GIT_COMMITTER_NAME`
* `GIT_COMMITTER_EMAIL`

The values can also be overridden by adding the appropriate flags. Run with `-help` to get the details.

If none of these is set, Semanticore will use `Semanticore Bot` as name and `semanticore@aoe.com` as E-Mail for Author and Committer.

### Configure filename of changelog

To configure the name of the changelog file, you can use the `CHANGELOG_FILE_NAME`. environment variable. If this variable is not set,
the default value `Changelog.md` will be used.

### Change label synchronization

Semanticore can propagate a `change::...` label to the release MR/PR.

This feature is disabled by default and only becomes active when both settings are configured:

* `SEMANTICORE_CHANGE_LABELS_ENABLED=true`
* `SEMANTICORE_CHANGE_LABELS` with a CSV list of **at least two** full labels in priority order
  (highest priority first)
* `SEMANTICORE_CHANGE_LABEL_DEFAULT` – label returned when nothing matches at all;
  must be contained in `SEMANTICORE_CHANGE_LABELS`

The default fallback is optional. When not set, no label is added for the no-match case.

Example:

* `SEMANTICORE_CHANGE_LABELS=change::emergency,change::major,change::normal,change::standard`

You can optionally map semantic commit types to change labels:

* `SEMANTICORE_CHANGE_LABEL_MAP=feat=change::normal,chore=change::standard,ops=change::standard`

Use the map to define feature fallback behavior, e.g. map `feat` to `change::normal`.

Each mapped label must be part of `SEMANTICORE_CHANGE_LABELS`, otherwise the feature is disabled.

When active, Semanticore always synchronizes only labels starting with `change::` and keeps all other labels untouched.

## Using Semanticore

To test Semanticore locally you can run it without an API token to create an example Changelog:

```
go run github.com/aoepeople/semanticore@v0 <optional path to repository>
```

### Example Configurations

#### Github Action

`.github/workflows/semanticore.yml`
```yaml
name: Semanticore

on:
  push:
    branches:
      - main
jobs:
  semanticore:
    runs-on: ubuntu-latest
    name: Semanticore
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.*'
      - name: Semanticore
        run: go run github.com/aoepeople/semanticore@v0
        env:
          SEMANTICORE_TOKEN: ${{secrets.GITHUB_TOKEN}}
          GOTOOLCHAIN: auto
```

#### Gitlab CI

Create a secret `SEMANTICORE_TOKEN` containing an API token with `api` and `write_repository` scope.

`.gitlab-ci.yml`
```yaml
stages:
  - semanticore

semanticore:
  image: golang:1
  stage: semanticore
  variables:
    GOTOOLCHAIN: auto
  script:
    - go run github.com/aoepeople/semanticore@v0
  only:
    - main
```

Make sure you set the repositories clone depth too a large enough value, the default of `50` might be too low.
