package internal

import (
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
)

func TestReadReposito(t *testing.T) {
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

	vhash = testCommit("Release v0.0.3")
	repository, err = ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	assert.Equal(t, "v0.0.3", repository.Latest)
	assert.Equal(t, vhash.String(), repository.unreleased)

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
