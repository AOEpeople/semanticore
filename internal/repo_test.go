package internal

import (
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

type testBackend struct {
	tag       string
	ref       string
	changelog string
}

func (*testBackend) String() string { return "testBackend" }
func (*testBackend) Name() string   { return "testBackend" }
func (b *testBackend) Release(tag, ref, changelog string) error {
	b.tag = tag
	b.ref = ref
	b.changelog = changelog
	return nil
}
func (*testBackend) MergeRequest(target, title, description, labels string) error { return nil }
func (*testBackend) CloseMergeRequest() error                                     { return nil }
func (*testBackend) MainBranch() (string, error)                                  { return "main", nil }

func TestReadRepository(t *testing.T) {
	mockRepo, err := git.Init(memory.NewStorage(), memfs.New())
	assert.NoError(t, err)

	cfg, err := mockRepo.Config()
	assert.NoError(t, err)
	cfg.Author.Email = "testing@example.com"
	cfg.Author.Name = "testing"
	cfg.User.Email = "testing@example.com"
	cfg.User.Name = "testing"
	err = mockRepo.SetConfig(cfg)
	assert.NoError(t, err)

	mockRepo.CreateBranch(&config.Branch{Name: "main"})
	mockWt, err := mockRepo.Worktree()
	assert.NoError(t, err)

	mockWt.Checkout(&git.CheckoutOptions{Branch: "main"})

	_, err = ReadRepository(mockRepo, true)
	assert.Error(t, err)

	file, err := mockWt.Filesystem.Create("test.file")
	assert.NoError(t, err)

	testCommit := func(msg string) plumbing.Hash {
		file.Write([]byte("msg"))
		mockWt.Add("test.file")
		hash, err := mockWt.Commit(msg, &git.CommitOptions{})
		assert.NoError(t, err)
		return hash
	}

	testCommit("test(semanticore): initial commit")

	repository, err := ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Equal(t, "", repository.unreleased)
	assert.Equal(t, "", repository.unreleasedChangelog)
	assert.Len(t, repository.tests, 1)

	vhash := testCommit("ci(semanticore): initial ci")
	mockRepo.CreateTag("v0.0.1", vhash, nil)
	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.1", repository.Latest)
	assert.Equal(t, "", repository.unreleased)

	vhash = testCommit("ci(semanticore): initial ci")
	mockRepo.CreateTag("v0.0.2", vhash, &git.CreateTagOptions{Message: "v0.0.2"})
	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.2", repository.Latest)
	assert.Equal(t, "", repository.changelog)

	cf, err := mockWt.Filesystem.Create("Changelog.md")
	assert.NoError(t, err)
	defer cf.Close()
	cf.Write([]byte(`## Version 1.2.3 test ## Version 1.2.3 ## Version 1.2.3`))
	mockWt.Add("Changelog.md")
	vhash = testCommit("Release v0.0.3")
	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.3", repository.Latest)
	assert.Equal(t, vhash.String(), repository.unreleased)
	assert.Equal(t, "## Version 1.2.3 test", repository.unreleasedChangelog)
	testBackend := new(testBackend)
	assert.NoError(t, repository.Release(testBackend))
	assert.Equal(t, "## Version 1.2.3 test", testBackend.changelog)

	testCommit("ci(semanticore): next ci")
	testCommit("test(semanticore): next test")
	testCommit("chore(semanticore): initial chore")
	testCommit("docs(semanticore): initial docs")
	testCommit("perf(semanticore): initial perf")
	testCommit("refactor(semanticore): initial refactor")
	testCommit("security(semanticore): initial security")
	testCommit("initial something whatever")
	testCommit("task: initial task")

	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Len(t, repository.tests, 1)
	assert.Len(t, repository.ops, 1)
	assert.Equal(t, 0, repository.Major)
	assert.Equal(t, 0, repository.Minor)
	assert.Equal(t, 4, repository.Patch)

	testCommit("feat(semanticore): initial feature")

	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Len(t, repository.tests, 1)
	assert.Len(t, repository.ops, 1)
	assert.Equal(t, 0, repository.Major)
	assert.Equal(t, 1, repository.Minor)
	assert.Equal(t, 0, repository.Patch)

	testCommit("feat(semanticore): second feature")

	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Len(t, repository.tests, 1)
	assert.Len(t, repository.ops, 1)
	assert.Equal(t, 0, repository.Major)
	assert.Equal(t, 1, repository.Minor)
	assert.Equal(t, 0, repository.Patch)

	testCommit("fix(semanticore): initial fix")
	testCommit("fix(semanticore): second fix")

	testCommit("fix(semanticore)!: final fix")

	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Len(t, repository.tests, 1)
	assert.Len(t, repository.ops, 1)
	assert.Len(t, repository.fixes, 3)
	assert.Equal(t, 1, repository.Major)
	assert.Equal(t, 0, repository.Minor)
	assert.Equal(t, 0, repository.Patch)

	repository, err = ReadRepository(mockRepo, false)
	assert.NoError(t, err)
	assert.Len(t, repository.tests, 1)
	assert.Len(t, repository.ops, 1)
	assert.Len(t, repository.fixes, 3)
	assert.Equal(t, 0, repository.Major)
	assert.Equal(t, 1, repository.Minor)
	assert.Equal(t, 0, repository.Patch)
}

// Tags that merely contain a long run of digits, like a ticket id in archive/123456-feature,
// must not be read as a version. The unescaped dots of the old regex matched any character.
func TestReadRepositoryIgnoresTagsWithoutDottedVersion(t *testing.T) {
	cases := []struct {
		name string
		tag  string
	}{
		{"ticket id in a prefixed name", "archive/123456-feature"},
		{"five digit id", "archive/12345-feature"},
		{"seven digit id", "archive/1234567-feature"},
		{"bare id", "54321"},
		{"version with a missing dot", "1.2-123456"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockRepo, err := git.Init(memory.NewStorage(), memfs.New())
			assert.NoError(t, err)
			mockWt, err := mockRepo.Worktree()
			assert.NoError(t, err)
			file, err := mockWt.Filesystem.Create("test.file")
			assert.NoError(t, err)

			start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			testCommit := func(msg string, minutes int) plumbing.Hash {
				file.Write([]byte(msg))
				mockWt.Add("test.file")
				sig := &object.Signature{Name: "testing", Email: "testing@example.com", When: start.Add(time.Duration(minutes) * time.Minute)}
				hash, err := mockWt.Commit(msg, &git.CommitOptions{Author: sig, Committer: sig})
				assert.NoError(t, err)
				return hash
			}

			_, err = mockRepo.CreateTag("2.5.1", testCommit("chore: first", 0), nil)
			assert.NoError(t, err)
			_, err = mockRepo.CreateTag(c.tag, testCommit("chore: second", 1), nil)
			assert.NoError(t, err)

			repository, err := ReadRepository(mockRepo, true)
			assert.NoError(t, err)
			assert.Equal(t, "2.5.1", repository.Latest)
			assert.Len(t, repository.chores, 1)
		})
	}
}
