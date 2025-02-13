# Changelog

## Version v0.5.2 (2023-10-31)

### Fixes

- **deps:** update module github.com/go-git/go-git/v5 to v5.10.0 (#66) (9d5cfbda)
- **deps:** update module github.com/go-git/go-git/v5 to v5.9.0 (d6b928de)
- **deps:** update module github.com/go-git/go-billy/v5 to v5.5.0 (21ad20a4)
- **deps:** update module github.com/stretchr/testify to v1.8.4 (#62) (d19b7fc4)
- **deps:** update module github.com/go-git/go-git/v5 to v5.5.2 (701140f0)
- **deps:** update module github.com/go-git/go-git/v5 to v5.5.0 (cfd9217e)

### Ops and CI/CD

- **github:** remove matrix strategy (3fbb0b15)

### Chores and tidying

- **deps:** update actions/checkout action to v4 (108a7c2f)
- **deps:** update actions/setup-go action to v4 (d0777ccd)

## Version v0.5.1 (2022-11-22)

### Fixes

- let fallback helper return the actual value (d3528fcb)

## Version v0.5.0 (2022-11-21)

### Features

- Allow to configure committer mail and name (ea4ab630)

### Fixes

- **deps:** update module github.com/stretchr/testify to v1.8.1 (aa5d09a1)
- **deps:** update module github.com/stretchr/testify to v1.8.0 (9f545314)

### Chores and tidying

- **deps:** update module go to 1.19 (c9530fde)
- **deps:** update irongut/codecoveragesummary action to v1.3.0 (481e6255)

## Version v0.4.0 (2022-06-14)

### Features

- **cli:** add backend flag to allow configuration if autodetection doesn't work (ada14bf7)

### Fixes

- **deps:** update module github.com/stretchr/testify to v1.7.2 (67a18a1c)

## Version v0.3.2 (2022-05-17)

### Fixes

- **release:** include changelog in release notes (721da6d2)

### Tests

- **release:** unit test release process (178a336d)

## Version v0.3.1 (2022-05-13)

### Fixes

- **changelog:** do not generate empty changelogs (03a5acc3)

## Version v0.3.0 (2022-05-13)

### Features

- **npm:** update version field in package.json (88dcf46c)

### Fixes

- **cli:** keep local commit (a510b6c2)

### Refactoring

- **semanticore:** move code to internal and add tests (b729eae9)
- **semanticore:** smaller code adoptions (0d37b5dc)

## Version v0.2.6 (2022-04-12)

### Fixes

- **changelog:** special character encoding (6d9b377b)

## Version v0.2.5 (2022-03-22)

### Fixes

- **gitlab:** search only for release branch in mr (a749aa6e)
- **gitlab:** search only for release branch in mr (49ccf332)

## Version v0.2.4 (2022-03-18)

### Fixes

- **versions:** default use v prefix (b6f5a7d6)
- **deps:** update module github.com/stretchr/testify to v1.7.1 (74288dfe)

## Version v0.2.3 (2022-03-14)

### Fixes

- avoid nil pointer for repos without existing tags (59be6d2e)

## Version v0.2.2 (2022-03-14)

### Fixes

- **changelog:** always include vPrefix (a056ad8e)

## Version 0.2.1 (2022-03-14)

### Fixes

- **log:** use BFS with additional ancestor check (33bcfc1a)

## Version 0.2.0 (2022-03-14)

### Features

- **versions:** add -no-v-prefix to remove v on versions (a723f8bc)

### Fixes

- **cli:** remove unused tag flag (a7c61016)
- **tags:** error in condition (49259f24)
- **commit message:** parse more exotic commits (ec033bd2)
- **semanticore:** do not commit without SEMANTICORE_TOKEN (5b8588ae)
- **commit message:** include semanticore link (e47e6d43)

### Ops and CI/CD

- **semanticore:** do not rely on cached dev versions in CI (2e1163d5)

### Documentation

- **readme:** add table of commit types (2518e4c2)

## Version v0.1.3 (2022-03-07)

### Fixes

- **tags:** correctly handle lightweight and annotated tags (6dd9a41e)
- **versions:** correctly parse existing version tags (#13) (24f407a5)
- **merge-request:** correctly show major release (#11) (2f7d6305)

## Version v0.1.2 (2022-03-04)

### Fixes

- **release-commits:** parse merge requests with one line (013ec753)
- **releases:** include changelog in docs (#9) (1c357053)

### Documentation

- **semanticore:** suggest using v0 instead of main (0679b50b)

## Version v0.1.1 (2022-03-04)

### Fixes

- **commitparser:** correctly identify release commits (6f8cd9a8)
- **github:** correctly parse github release commits (cf296de5)

## Version v0.1.0 (2022-03-04)

### Features

- **semanticore:** release semanticore (ff66964d)

### Fixes

- **github:** use correct method for closing PRs (e382ebb0)

### Ops and CI/CD

- **github:** fetch history for semanticore job (af2ae333)
- **github:** fetch history (829c1011)

### Documentation

- **github:** use actions v3 (e324eea0)

### Chores and tidying

- **deps:** update actions/setup-go action to v3 (#7) (da2a0467)
- **deps:** update actions/checkout action to v3 (#6) (8a5c4115)
