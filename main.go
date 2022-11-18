package main

import (
	"bytes"
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

var useBackend = flag.String("backend", os.Getenv("SEMANTICORE_BACKEND"), "configure backend use either \"github\" or \"gitlab\" - we'll try to autodetect if empty")
var createMajor = flag.Bool("major", false, "release major versions")
var createRelease = flag.Bool("release", true, "create release alongside tags")
var createMergeRequest = flag.Bool("merge-request", true, "create merge release for branch")
var authorName = flag.String("git-author-name", emptyFallback(os.Getenv("GIT_AUTHOR_NAME"), "Semanticore Bot"), "author name for the git commits, falls back to env var GIT_AUTHOR_NAME and afterwards to \"Semanticore Bot\"")
var authorEmail = flag.String("git-author-email", emptyFallback(os.Getenv("GIT_AUTHOR_EMAIL"), "semanticore@aoe.com"), "author email for the git commits, falls back to env var GIT_AUTHOR_EMAIL and afterwards to \"semanticore@aoe.com\"")
var committerName = flag.String("git-committer-name", emptyFallback(os.Getenv("GIT_COMMITTER_NAME"), "Semanticore Bot"), "committer name for the git commits, falls back to env var GIT_COMMITTER_NAME and afterwards to \"Semanticore Bot\"")
var committerEmail = flag.String("git-committer-email", emptyFallback(os.Getenv("GIT_COMMITTER_EMAIL"), "semanticore@aoe.com"), "committer email for the git commits, falls back to env var GIT_COMMITTER_EMAIL and afterwards to \"semanticore@aoe.com\"")

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

	repository, err := internal.ReadRepository(repo, *createMajor)
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

	filename := "Changelog.md"
	files, err := wt.Filesystem.ReadDir(".")
	try(err)

	// detect case-sensitive filenames
	for _, f := range files {
		if !f.IsDir() && strings.ToLower(f.Name()) == "changelog.md" {
			filename = f.Name()
		}
	}

	cl, _ := os.ReadFile(filepath.Join(filename))
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

	commit, err := wt.Commit(fmt.Sprintf("Release %s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch), &git.CommitOptions{
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
	})
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
	return ""
}
