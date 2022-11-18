package hook

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"

	"github.com/aoepeople/semanticore/internal"
)

func TestNpmUpdateVersionHook(t *testing.T) {
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

	file, err := mockWt.Filesystem.Create("test.file")
	assert.NoError(t, err)

	testCommit := func(msg string) plumbing.Hash {
		file.Write([]byte("msg"))
		mockWt.Add("test.file")
		hash, err := mockWt.Commit(msg, &git.CommitOptions{})
		assert.NoError(t, err)
		return hash
	}

	packagejson = "package.json"

	packagejson, err := mockWt.Filesystem.Create("package.json")
	assert.NoError(t, err)
	_, err = packagejson.Write([]byte(`{"foo": "bar",    "version": 	"1.2.3" , "dependencies": {"foo": "1.2.3"}  }`))
	assert.NoError(t, err)
	packagejson.Close()
	mockWt.Add("package.json")
	testCommit("test(semanticore): initial commit")

	repository, err := internal.ReadRepository(mockRepo, true)
	assert.NoError(t, err)
	repository.Major = 4
	repository.Minor = 5
	repository.Patch = 6
	NpmUpdateVersionHook(mockWt, repository)

	packagejson, err = mockWt.Filesystem.Open("package.json")
	assert.NoError(t, err)
	b, _ := io.ReadAll(packagejson)
	var jsonData struct {
		Version      string `json:"version"`
		Dependencies struct {
			Foo string `json:"foo"`
		} `json:"dependencies"`
	}
	assert.NoError(t, json.Unmarshal(b, &jsonData), "can not read json: %s", b)
	packagejson.Close()
	assert.Equal(t, "4.5.6", jsonData.Version, "json content does not match: %s", b)
	assert.Equal(t, "1.2.3", jsonData.Dependencies.Foo, "dependency version was updated: %s", b)
}
