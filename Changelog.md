# Changelog

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

