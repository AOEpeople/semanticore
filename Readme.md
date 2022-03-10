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
    strategy:
      matrix:
        go: [ '1.*' ]
    name: Semanticore
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: ${{ matrix.go }}
      - name: Semanticore
        run: go run github.com/aoepeople/semanticore@v0
        env:
          SEMANTICORE_TOKEN: ${{secrets.GITHUB_TOKEN}}
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
  script:
    - go run github.com/aoepeople/semanticore@v0
  only:
    - main
```
