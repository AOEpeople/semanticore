package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/aoepeople/semanticore/internal"
	"github.com/aoepeople/semanticore/internal/hook"
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	useBackend         = flag.String("backend", os.Getenv("SEMANTICORE_BACKEND"), "configure backend use either \"github\" or \"gitlab\" - we'll try to autodetect if empty")
	createMajor        = flag.Bool("major", false, "release major versions")
	createRelease      = flag.Bool("release", true, "create release alongside tags")
	createMergeRequest = flag.Bool("merge-request", true, "create merge release for branch")
	authorName         = flag.String("git-author-name", emptyFallback(os.Getenv("GIT_AUTHOR_NAME"), "Semanticore Bot"), "author name for the git commits, falls back to env var GIT_AUTHOR_NAME and afterwards to \"Semanticore Bot\"")
	authorEmail        = flag.String("git-author-email", emptyFallback(os.Getenv("GIT_AUTHOR_EMAIL"), "semanticore@aoe.com"), "author email for the git commits, falls back to env var GIT_AUTHOR_EMAIL and afterwards to \"semanticore@aoe.com\"")
	committerName      = flag.String("git-committer-name", emptyFallback(os.Getenv("GIT_COMMITTER_NAME"), "Semanticore Bot"), "committer name for the git commits, falls back to env var GIT_COMMITTER_NAME and afterwards to \"Semanticore Bot\"")
	committerEmail     = flag.String("git-committer-email", emptyFallback(os.Getenv("GIT_COMMITTER_EMAIL"), "semanticore@aoe.com"), "committer email for the git commits, falls back to env var GIT_COMMITTER_EMAIL and afterwards to \"semanticore@aoe.com\"")
	changelogMaxLines  = flag.Int("changelog-max-lines", 0, "trim the changelog to the last version including the maximum configured lines")
	changelogFileName  = flag.String("changelog-file-name", emptyFallback(os.Getenv("CHANGELOG_FILE_NAME"), "Changelog.md"), "filename for changelog, falls back to env var CHANGELOG_FILE_NAME and afterwards to \"Changelog.md\"")
	signKeyFilePath    = flag.String("sign-key-file", emptyFallback(os.Getenv("SEMANTICORE_SIGN_KEY_FILE"), ""), "path to GPG private key file for signing commits")
	changeLabelsEnabled        = flag.Bool("change-labels-enabled", strings.EqualFold(os.Getenv("SEMANTICORE_CHANGE_LABELS_ENABLED"), "true"), "enable change label sync for merge requests")
	changeLabels               = flag.String("change-labels", emptyFallback(os.Getenv("SEMANTICORE_CHANGE_LABELS"), ""), "CSV list of labels in priority order (highest first), e.g. change::emergency,change::major,change::normal,change::standard")
	changeLabelMap             = flag.String("change-label-map", emptyFallback(os.Getenv("SEMANTICORE_CHANGE_LABEL_MAP"), ""), "CSV list mapping semantic commit types to labels, e.g. feat=change::normal,chore=change::standard")
	changeLabelDefault         = flag.String("change-label-default", emptyFallback(os.Getenv("SEMANTICORE_CHANGE_LABEL_DEFAULT"), ""), "label to use when no commit label matches; must be in --change-labels")
)

func main() {
	flag.Parse()

	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	try(os.Chdir(dir))

	repo, err := git.PlainOpen(".")
	try(err)

	remote, err := repo.Remote("origin")
	try(err)
	remoteUrl, err := url.Parse(remote.Config().URLs[0])
	try(err)
	repoId := strings.TrimSuffix(strings.TrimPrefix(remoteUrl.Path, "/"), ".git")
	log.Printf("[semanticore] repository: %s at %s", repoId, remoteUrl.Host)

	var backend internal.Backend
	if os.Getenv("SEMANTICORE_TOKEN") == "" {
		log.Println("[semanticore] SEMANTICORE_TOKEN unset, no merge requests will be handled")
	} else if *useBackend == "github" || remoteUrl.Host == "github.com" {
		backend = internal.NewGithubBackend(os.Getenv("SEMANTICORE_TOKEN"), repoId)
	} else if *useBackend == "gitlab" || strings.Contains(remoteUrl.Host, "gitlab") {
		backend = internal.NewGitlabBackend(os.Getenv("SEMANTICORE_TOKEN"), remoteUrl.Host, repoId)
	}

	head, err := repo.Head()
	try(err)

	// derive label prefix from configured labels when the feature is enabled
	labelPrefix := ""
	if *changeLabelsEnabled {
		if priority, ok := parseChangeLabels(*changeLabels); ok {
			labelPrefix = commonLabelPrefix(priority)
		}
	}

	repository, err := internal.ReadRepositoryWithPrefix(repo, *createMajor, labelPrefix)
	try(err)

	if backend != nil && *createRelease {
		repository.Release(backend)
	}

	changelog := repository.Changelog()

	if changelog == "" {
		log.Println("no changes detected, exiting...")
		return
	}

	fmt.Println(changelog)

	if !*createMergeRequest {
		return
	}

	wt, err := repo.Worktree()
	try(err)

	filename := *changelogFileName
	files, err := wt.Filesystem.ReadDir(".")
	try(err)

	// detect case-sensitive filenames
	for _, f := range files {
		if !f.IsDir() && strings.EqualFold(f.Name(), filename) {
			filename = f.Name()
		}
	}

	cl, _ := os.ReadFile(filepath.Join(filename))

	if *changelogMaxLines > 0 {
		cl = internal.TrimChangelog(cl, *changelogMaxLines)
	}

	if strings.Contains(string(cl), "# Changelog\n\n") {
		cl = bytes.Replace(cl, []byte("# Changelog\n\n"), []byte(changelog), 1)
	} else if strings.Contains(string(cl), "# Changelog\n") {
		cl = bytes.Replace(cl, []byte("# Changelog\n"), []byte(changelog), 1)
	} else {
		cl = append([]byte(changelog), cl...)
	}
	try(os.WriteFile(filepath.Join(filename), cl, 0644))

	_, err = wt.Add(filename)
	try(err)

	hook.NpmUpdateVersionHook(wt, repository)

	signKey, err := internal.TryCreateSignKey(signKeyFilePath)
	if errors.Is(err, internal.ErrNoSigningKeyFound) {
		log.Printf("[semanticore] no signing key found, commit will not be signed")
	} else if err != nil {
		try(err)
	}

	commitOptions := &git.CommitOptions{
		Author: &object.Signature{
			Name:  *authorName,
			Email: *authorEmail,
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  *committerName,
			Email: *committerEmail,
			When:  time.Now(),
		},
		SignKey: signKey,
	}

	commit, err := wt.Commit(fmt.Sprintf("Release %s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch), commitOptions)
	try(err)

	log.Printf("[semanticore] committed changelog: %s", commit.String())

	try(wt.Reset(&git.ResetOptions{
		Commit: head.Hash(),
		Mode:   git.HardReset,
	}))

	if backend == nil {
		log.Printf("no backend configured, keeping changes in a local commit: %s", commit.String())
		return
	}
	try(repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(commit.String() + ":refs/heads/semanticore/release")},
		Force:      true,
		Auth:       backend,
		Progress:   os.Stdout,
	}))

	releasetype := "patch 🩹"
	if repository.Breaking && *createMajor {
		releasetype = "major 👏"
	} else if len(repository.Features) > 0 {
		releasetype = "minor 📦"
	}
	labels := "Release 🏆," + releasetype
	if *changeLabelsEnabled {
		if priority, ok := parseChangeLabels(*changeLabels); ok {
			if backend != nil {
				repository.CollectIssuePrefixedLabels(backend, labelPrefix)
			}
			defaultLabel, defaultOk := parseChangeLabelDefault(*changeLabelDefault, priority)
			if !defaultOk {
				log.Printf("[semanticore] change labels are enabled but SEMANTICORE_CHANGE_LABEL_DEFAULT is not part of SEMANTICORE_CHANGE_LABELS, disabling feature")
			} else if semanticMap, ok := parseChangeLabelMap(*changeLabelMap, priority); ok {
				repository.ChangeLabel = repository.DetermineChangeLabel(priority, semanticMap, defaultLabel)
				labels += "," + repository.ChangeLabel
			} else {
				log.Printf("[semanticore] change labels are enabled but SEMANTICORE_CHANGE_LABEL_MAP is invalid, disabling feature")
			}
		} else {
			log.Printf("[semanticore] change labels are enabled but SEMANTICORE_CHANGE_LABELS is invalid, disabling feature")
		}
	}
	description := fmt.Sprintf(`# Release %s%d.%d.%d 🏆

## Summary

There are %s commits since %s.

This is a %s release.

Merge this pull request to commit the changelog and have Semanticore create a new release on the next pipeline run.

%s

---

This changelog was generated by your friendly [Semanticore Release Bot](https://github.com/aoepeople/semanticore)
`, repository.VPrefix, repository.Major, repository.Minor, repository.Patch, strings.Join(repository.Details, ", "), repository.Latest, releasetype, strings.TrimSpace(changelog))

	mainBranch, err := backend.MainBranch()
	try(err)

	try(backend.MergeRequest(string(mainBranch), fmt.Sprintf("Release %s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch), description, labels))
}

func emptyFallback(s, fallback string) string {
	if s == "" {
		return fallback
	}

	return s
}

func parseChangeLabels(raw string) ([]string, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) < 2 {
		return nil, false
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		label := strings.TrimSpace(strings.ToLower(part))
		if label == "" || !strings.Contains(label, "::") {
			return nil, false
		}
		if _, ok := seen[label]; ok {
			return nil, false
		}
		seen[label] = struct{}{}
		out = append(out, label)
	}

	return out, true
}

// commonLabelPrefix returns the longest common prefix shared by all labels,
// e.g. ["change::emergency","change::normal"] → "change::".
// Returns "" if no common prefix exists.
func commonLabelPrefix(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	prefix := labels[0]
	for _, label := range labels[1:] {
		for !strings.HasPrefix(label, prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

// parseChangeLabelDefault validates that defaultLabel is present in the
// priority list. Empty string is allowed (no default configured).
func parseChangeLabelDefault(defaultLabel string, priority []string) (string, bool) {
	allowed := map[string]struct{}{}
	for _, l := range priority {
		allowed[strings.ToLower(strings.TrimSpace(l))] = struct{}{}
	}
	dl := strings.ToLower(strings.TrimSpace(defaultLabel))
	if dl != "" {
		if _, ok := allowed[dl]; !ok {
			return "", false
		}
	}
	return dl, true
}

func parseChangeLabelMap(raw string, priority []string) (map[string]string, bool) {
	allowed := map[string]struct{}{}
	for _, label := range priority {
		allowed[strings.ToLower(strings.TrimSpace(label))] = struct{}{}
	}

	out := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return out, true
	}

	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, false
		}

		semanticType := strings.ToLower(strings.TrimSpace(parts[0]))
		label := strings.ToLower(strings.TrimSpace(parts[1]))
		if semanticType == "" || label == "" {
			return nil, false
		}
		if _, ok := allowed[label]; !ok {
			return nil, false
		}
		if _, ok := out[semanticType]; ok {
			return nil, false
		}

		out[semanticType] = label
	}

	return out, true
}

