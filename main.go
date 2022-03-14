package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aoepeople/semanticore/internal"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var createMajor = flag.Bool("major", false, "release major versions")
var createRelease = flag.Bool("release", true, "create release alongside tags")
var createMergeRequest = flag.Bool("merge-request", true, "create merge release for branch")

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
		log.Println("[semanticore] SEMANTICORE_TOKEN unset, no commits/changelog will be done")
	} else if remoteUrl.Host == "github.com" {
		backend = internal.NewGithubBackend(os.Getenv("SEMANTICORE_TOKEN"), repoId)
	} else if strings.Contains(remoteUrl.Host, "gitlab") {
		backend = internal.NewGitlabBackend(os.Getenv("SEMANTICORE_TOKEN"), remoteUrl.Host, repoId)
	}

	tags := make(map[string][]*plumbing.Reference)
	gittags, err := repo.Tags()
	try(err)

	err = gittags.ForEach(func(r *plumbing.Reference) error {
		tag, _ := repo.TagObject(r.Hash())
		if tag != nil {
			tags[tag.Target.String()] = append(tags[tag.Target.String()], r)
		} else {
			tags[r.Hash().String()] = append(tags[r.Hash().String()], r)
		}
		return nil
	})
	try(err)

	glog, err := repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
	try(err)

	major, minor, patch := 0, 0, 0
	vPrefix := ""
	vregex := regexp.MustCompile(`(v?)(\d+).(\d+).(\d+)`)
	var logs []*object.Commit
	glog.ForEach(func(c *object.Commit) error {
		if tags, ok := tags[c.Hash.String()]; ok {
			for _, tag := range tags {
				match := vregex.FindStringSubmatch(tag.Name().String())
				if match == nil {
					continue
				}
				tagMajor, _ := strconv.Atoi(match[2])
				tagMinor, _ := strconv.Atoi(match[3])
				tagPatch, _ := strconv.Atoi(match[4])
				if tagMajor > major || (tagMajor == major && tagMinor > minor) || (tagMajor == major && tagMinor == minor && tagPatch > patch) {
					major = tagMajor
					minor = tagMinor
					patch = tagPatch
					vPrefix = match[1]
					return errors.New("done")
				}
			}
		}
		logs = append(logs, c)
		return nil
	})

	latest := fmt.Sprintf("%s%d.%d.%d", vPrefix, major, minor, patch)
	log.Printf("[semanticore] Current version: %s", latest)

	reverst := regexp.MustCompile(`This reverts commit ([a-zA-Z0-9]+)`)
	_ = reverst

	reverted := make(map[string]struct{})

	var fixes []string
	var features []string
	var other []string
	var tests []string
	var chores []string
	var ops []string
	var docs []string
	var perf []string
	var refactor []string
	var security []string
	var releaseDate time.Time
	var breaking bool

	for _, commit := range logs {
		if _, ok := reverted[commit.Hash.String()]; ok {
			continue
		}
		msg := strings.TrimSpace(commit.Message)
		if match := reverst.FindStringSubmatch(msg); match != nil {
			reverted[match[1]] = struct{}{}
			continue
		}

		if newVprefix, newMajor, newMinor, newPatch := internal.DetectReleaseCommit(msg, len(commit.ParentHashes) > 1); newMajor+newMinor+newPatch > 0 {
			major = newMajor
			minor = newMinor
			patch = newPatch
			vPrefix = newVprefix
			latest = fmt.Sprintf("%s%d.%d.%d", vPrefix, major, minor, patch)
			log.Printf("[semanticore] found version %s at %s: %q", latest, commit.Hash, msg)
			if backend != nil && *createRelease {
				changelog := ""
				fi, err := commit.Files()
				if err == nil {
					fi.ForEach(func(f *object.File) error {
						if strings.ToLower(f.Name) == "changelog.md" {
							c, _ := f.Contents()
							changelog = "## Version " + strings.Split(c, "## Version ")[1]
							changelog = strings.TrimSpace(changelog)
						}
						return nil
					})
				}
				try(backend.Release(latest, commit.Hash.String(), changelog))
			}
			break
		}

		if len(commit.ParentHashes) > 1 {
			continue
		}
		if commit.Committer.When.After(releaseDate) {
			releaseDate = commit.Committer.When
		}
		typ, scope, msg, major := internal.ParseCommitMessage(msg)
		breaking = breaking || major
		line := fmt.Sprintf("%s (%s)", msg, commit.Hash.String()[:8])
		if scope != "" {
			line = fmt.Sprintf("**%s:** %s (%s)", scope, msg, commit.Hash.String()[:8])
		}
		switch typ {
		case internal.TypeFeat:
			features = append(features, line)
		case internal.TypeFix:
			fixes = append(fixes, line)
		case internal.TypeTest:
			tests = append(tests, line)
		case internal.TypeChore:
			chores = append(chores, line)
		case internal.TypeOps:
			ops = append(ops, line)
		case internal.TypeDocs:
			docs = append(docs, line)
		case internal.TypePerf:
			perf = append(perf, line)
		case internal.TypeRefactor:
			refactor = append(refactor, line)
		case internal.TypeSecurity:
			security = append(security, line)
		default:
			other = append(other, line)
		}
	}

	if len(features)+len(fixes)+len(tests)+len(chores)+len(ops)+len(docs)+len(perf)+len(refactor)+len(security)+len(other) == 0 {
		// no changes detected
		log.Println("[semanticore] no changes detected, no changelog created")
		if *createMergeRequest && backend != nil {
			try(backend.CloseMergeRequest())
		}
		return
	}

	if breaking && *createMajor {
		major++
		minor = 0
		patch = 0
	} else if len(features) > 0 {
		minor++
		patch = 0
	} else {
		patch++
	}

	changelog := fmt.Sprintf("# Changelog\n\n## Version %d.%d.%d (%s)\n\n", major, minor, patch, releaseDate.Format("2006-01-02"))

	changelogentries := []struct {
		title  string
		logs   []string
		detail string
	}{
		{"### Features", features, "🆕 feature"},
		{"### Security Fixes", security, "🚨 security"},
		{"### Fixes", fixes, "👾 fix"},
		{"### Tests", tests, "🛡 test"},
		{"### Refactoring", refactor, "🔁 refactor"},
		{"### Ops and CI/CD", ops, "🤖 devops"},
		{"### Documentation", docs, "📚 doc"},
		{"### Performance", perf, "⚡️ performance"},
		{"### Chores and tidying", chores, "🧹 chore"},
		{"### Other", other, "📝 other"},
	}
	var details []string
	for _, log := range changelogentries {
		if len(log.logs) < 1 {
			continue
		}
		changelog += fmt.Sprintln(log.title)
		changelog += fmt.Sprintln()
		for _, line := range log.logs {
			changelog += fmt.Sprintln("- " + line)
		}
		changelog += fmt.Sprintln()
		details = append(details, fmt.Sprintf("%d %s", len(log.logs), log.detail))
	}

	fmt.Println(changelog)

	if backend == nil || !*createMergeRequest {
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

	cl, _ := ioutil.ReadFile(filepath.Join(filename))
	if !strings.Contains(string(cl), "# Changelog") {
		cl = append([]byte(changelog), cl...)
	} else {
		cl = bytes.Replace(cl, []byte("# Changelog\n\n"), []byte(changelog), 1)
	}
	try(ioutil.WriteFile(filepath.Join(filename), cl, 0644))

	_, err = wt.Add(filename)
	try(err)

	head, err := repo.Head()
	try(err)
	commit, err := wt.Commit(fmt.Sprintf("Release %s%d.%d.%d", vPrefix, major, minor, patch), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Semanticore Bot",
			Email: "semanticore@aoe.com",
			When:  time.Now(),
		},
		Committer: &object.Signature{
			Name:  "Semanticore Bot",
			Email: "semanticore@aoe.com",
			When:  time.Now(),
		},
	})
	try(err)

	log.Printf("[semanticore] committed changelog: %s", commit.String())

	try(wt.Reset(&git.ResetOptions{
		Commit: head.Hash(),
		Mode:   git.HardReset,
	}))

	try(repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec(commit.String() + ":refs/heads/semanticore/release")},
		Force:      true,
		Auth:       backend,
		Progress:   os.Stdout,
	}))

	releasetype := "patch 🩹"
	if breaking && *createMajor {
		releasetype = "major 👏"
	} else if len(features) > 0 {
		releasetype = "minor 📦"
	}
	labels := "Release 🏆," + releasetype
	description := fmt.Sprintf(`# Release %d.%d.%d 🏆

## Summary

There are %s commits since %s.

This is a %s release.

%s

---

This changelog was generated by your friendly [Semanticore Release Bot](https://github.com/aoepeople/semanticore)
`, major, minor, patch, strings.Join(details, ", "), latest, releasetype, strings.TrimSpace(changelog))

	mainBranch, err := backend.MainBranch()
	try(err)

	try(backend.MergeRequest(string(mainBranch), fmt.Sprintf("Release %d.%d.%d", major, minor, patch), description, labels))
}
