package internal

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

type Repository struct {
	Major, Minor, Patch int
	VPrefix             string
	Latest              string

	fixes       []string
	Features    []string
	other       []string
	tests       []string
	chores      []string
	ops         []string
	docs        []string
	perf        []string
	refactor    []string
	security    []string
	releaseDate time.Time
	Breaking    bool
	Details     []string

	changelog string

	unreleased          string
	unreleasedChangelog string
}

func ReadRepository(repo *git.Repository, createMajor bool) (*Repository, error) {
	repository := &Repository{
		VPrefix: "v",
	}

	tags := make(map[string][]*plumbing.Reference)
	gittags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("unable to read repo tags: %w", err)
	}

	err = gittags.ForEach(func(r *plumbing.Reference) error {
		tag, _ := repo.TagObject(r.Hash())
		if tag != nil {
			tags[tag.Target.String()] = append(tags[tag.Target.String()], r)
		} else {
			tags[r.Hash().String()] = append(tags[r.Hash().String()], r)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to iterate git tags: %w", err)
	}

	glog, err := repo.Log(&git.LogOptions{
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to read repository log: %w", err)
	}

	vregex := regexp.MustCompile(`(v?)(\d+).(\d+).(\d+)`)
	var ancestor *object.Commit
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
				if tagMajor > repository.Major || (tagMajor == repository.Major && tagMinor > repository.Minor) || (tagMajor == repository.Major && tagMinor == repository.Minor && tagPatch > repository.Patch) {
					repository.Major = tagMajor
					repository.Minor = tagMinor
					repository.Patch = tagPatch
					repository.VPrefix = match[1]
					ancestor = c
					return errors.New("done")
				}
			}
		}
		return nil
	})

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("repo.Head() failed :%w", err)
	}

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("unable to read head commit :%w", err)
	}

	var ignore []plumbing.Hash
	var seen map[plumbing.Hash]bool
	if ancestor != nil {
		ignore = append(ignore, ancestor.Hash)
		seen = map[plumbing.Hash]bool{ancestor.Hash: true}
	}

	var logs []*object.Commit
	object.NewCommitIterBSF(headCommit, seen, ignore).ForEach(func(c *object.Commit) error {
		if a, _ := c.IsAncestor(ancestor); a {
			return storer.ErrStop
		}
		logs = append(logs, c)
		return nil
	})

	repository.Latest = fmt.Sprintf("%s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch)
	log.Printf("[semanticore] Current version: %s", repository.Latest)

	reverst := regexp.MustCompile(`This reverts commit ([a-zA-Z0-9]+)`)
	_ = reverst

	reverted := make(map[string]struct{})
	updates := 0

	for _, commit := range logs {
		if _, ok := reverted[commit.Hash.String()]; ok {
			continue
		}
		msg := strings.TrimSpace(commit.Message)
		if match := reverst.FindStringSubmatch(msg); match != nil {
			reverted[match[1]] = struct{}{}
			continue
		}

		if newVprefix, newMajor, newMinor, newPatch := DetectReleaseCommit(msg, len(commit.ParentHashes) > 1); newMajor+newMinor+newPatch > 0 {
			repository.Major = newMajor
			repository.Minor = newMinor
			repository.Patch = newPatch
			repository.VPrefix = newVprefix
			repository.Latest = fmt.Sprintf("%s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch)
			log.Printf("[semanticore] found version %s at %s: %q", repository.Latest, commit.Hash, msg)

			repository.unreleased = commit.Hash.String()

			changelog := ""
			fi, err := commit.Files()
			if err == nil {
				fi.ForEach(func(f *object.File) error {
					if strings.ToLower(f.Name) == "changelog.md" {
						c, _ := f.Contents()
						repository.unreleasedChangelog = "## Version " + strings.Split(c, "## Version ")[1]
						repository.unreleasedChangelog = strings.TrimSpace(changelog)
					}
					return nil
				})
			}

			break
		}

		if len(commit.ParentHashes) > 1 {
			continue
		}
		if commit.Committer.When.After(repository.releaseDate) {
			repository.releaseDate = commit.Committer.When
		}
		typ, scope, msg, major := ParseCommitMessage(msg)
		repository.Breaking = repository.Breaking || major
		line := fmt.Sprintf("%s (%s)", msg, commit.Hash.String()[:8])
		if scope != "" {
			line = fmt.Sprintf("**%s:** %s (%s)", scope, msg, commit.Hash.String()[:8])
		}
		switch typ {
		case TypeFeat:
			repository.Features = append(repository.Features, line)
		case TypeFix:
			repository.fixes = append(repository.fixes, line)
		case TypeTest:
			repository.tests = append(repository.tests, line)
		case TypeChore:
			repository.chores = append(repository.chores, line)
		case TypeOps:
			repository.ops = append(repository.ops, line)
		case TypeDocs:
			repository.docs = append(repository.docs, line)
		case TypePerf:
			repository.perf = append(repository.perf, line)
		case TypeRefactor:
			repository.refactor = append(repository.refactor, line)
		case TypeSecurity:
			repository.security = append(repository.security, line)
		default:
			repository.other = append(repository.other, line)
		}
		updates++
	}

	if updates == 0 {
		return repository, nil
	}

	if repository.Breaking && createMajor {
		repository.Major++
		repository.Minor = 0
		repository.Patch = 0
	} else if len(repository.Features) > 0 {
		repository.Minor++
		repository.Patch = 0
	} else {
		repository.Patch++
	}

	repository.changelog = fmt.Sprintf("# Changelog\n\n## Version %s%d.%d.%d (%s)\n\n", repository.VPrefix, repository.Major, repository.Minor, repository.Patch, repository.releaseDate.Format("2006-01-02"))

	changelogentries := []struct {
		title  string
		logs   []string
		detail string
	}{
		{"### Features", repository.Features, "🆕 feature"},
		{"### Security Fixes", repository.security, "🚨 security"},
		{"### Fixes", repository.fixes, "👾 fix"},
		{"### Tests", repository.tests, "🛡 test"},
		{"### Refactoring", repository.refactor, "🔁 refactor"},
		{"### Ops and CI/CD", repository.ops, "🤖 devops"},
		{"### Documentation", repository.docs, "📚 doc"},
		{"### Performance", repository.perf, "⚡️ performance"},
		{"### Chores and tidying", repository.chores, "🧹 chore"},
		{"### Other", repository.other, "📝 other"},
	}

	for _, log := range changelogentries {
		if len(log.logs) < 1 {
			continue
		}
		repository.changelog += fmt.Sprintln(log.title)
		repository.changelog += fmt.Sprintln()
		for _, line := range log.logs {
			repository.changelog += fmt.Sprintln("- " + line)
		}
		repository.changelog += fmt.Sprintln()
		repository.Details = append(repository.Details, fmt.Sprintf("%d %s", len(log.logs), log.detail))
	}

	return repository, nil
}

func (repository *Repository) Release(backend Backend) error {
	if err := backend.Release(repository.Latest, repository.unreleased, repository.unreleasedChangelog); err != nil {
		return fmt.Errorf("unable to release %s at %s: %w", repository.Latest, repository.unreleased, err)
	}
	return nil
}

func (repository *Repository) Changelog() string {
	return repository.changelog
}

func (repository *Repository) Version() string {
	return fmt.Sprintf("%s%d.%d.%d", repository.VPrefix, repository.Major, repository.Minor, repository.Patch)
}
