package internal

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
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
func (*testBackend) MergeRequest(_, _, _, _ string) error              { return nil }
func (*testBackend) CloseMergeRequest() error                          { return nil }
func (*testBackend) MainBranch() (string, error)                       { return "main", nil }
func (*testBackend) IssuePrefixedLabels(_ int, _ string) ([]string, error) { return nil, nil }
func (*testBackend) SetAuth(_ *http.Request)                           {}

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

func TestCollectIssuePrefixedLabels(t *testing.T) {
	priority := []string{"change::emergency", "change::major", "change::normal", "change::standard"}

	backend := &issueBackend{labels: map[int][]string{
		42:  {"change::major", "other-label"},
		123: {"change::normal"},
		99:  nil,
	}}

	repo := &Repository{
		changeLabels: map[string]struct{}{},
		issueRefs:    []int{42, 123, 99, 999},
	}
	repo.CollectIssuePrefixedLabels(backend, "change::")

	assert.Equal(t, "change::major", repo.DetermineChangeLabel(priority, nil, "change::standard"))
}

type issueBackend struct {
	labels map[int][]string
}

func (*issueBackend) String() string                                    { return "issueBackend" }
func (*issueBackend) Name() string                                      { return "issueBackend" }
func (*issueBackend) SetAuth(_ *http.Request)                           {}
func (*issueBackend) Release(_, _, _ string) error                     { return nil }
func (*issueBackend) MergeRequest(_, _, _, _ string) error             { return nil }
func (*issueBackend) CloseMergeRequest() error                         { return nil }
func (*issueBackend) MainBranch() (string, error)                      { return "main", nil }
func (b *issueBackend) IssuePrefixedLabels(id int, prefix string) ([]string, error) {
	labels, ok := b.labels[id]
	if !ok {
		return nil, fmt.Errorf("issue %d not found", id)
	}
	var out []string
	for _, label := range labels {
		if strings.HasPrefix(strings.ToLower(label), strings.ToLower(prefix)) {
			out = append(out, label)
		}
	}
	return out, nil
}

func TestDetermineChangeLabel(t *testing.T) {
	priority := []string{"change::emergency", "change::major", "change::normal", "change::standard"}
	dl := "change::standard"

	// explicit label match
	repo := &Repository{changeLabels: map[string]struct{}{"change::major": {}}}
	assert.Equal(t, "change::major", repo.DetermineChangeLabel(priority, nil, dl))

	// no feature mapping -> default
	repo = &Repository{Features: []string{"new feat"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::standard", repo.DetermineChangeLabel(priority, nil, dl))

	// feature mapping via semantic map
	repo = &Repository{Features: []string{"new feat"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::normal", repo.DetermineChangeLabel(priority, map[string]string{"feat": "change::normal"}, dl))

	// no match → default
	repo = &Repository{changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::standard", repo.DetermineChangeLabel(priority, nil, dl))

	// no default configured → empty string
	repo = &Repository{Features: []string{"x"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "", repo.DetermineChangeLabel(priority, nil, ""))

	// semantic map match
	repo = &Repository{chores: []string{"chore message"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::standard", repo.DetermineChangeLabel(priority, map[string]string{"chore": "change::standard"}, dl))

	// explicit label beats lower-priority semantic map entry
	repo = &Repository{ops: []string{"ops message"}, changeLabels: map[string]struct{}{"change::major": {}}}
	assert.Equal(t, "change::major", repo.DetermineChangeLabel(priority, map[string]string{"ops": "change::standard"}, dl))

	// 3-label variant with explicit default + map
	priority3 := []string{"change::emergency", "change::normal", "change::standard"}
	repo = &Repository{changeLabels: map[string]struct{}{"change::emergency": {}}}
	assert.Equal(t, "change::emergency", repo.DetermineChangeLabel(priority3, nil, "change::standard"))
	repo = &Repository{Features: []string{"x"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::normal", repo.DetermineChangeLabel(priority3, map[string]string{"feat": "change::normal"}, "change::standard"))
	repo = &Repository{changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::standard", repo.DetermineChangeLabel(priority3, nil, "change::standard"))

	// 5-label variant with custom mapping + explicit default
	priority5 := []string{"change::emergency", "change::major", "change::normal", "change::minor", "change::standard"}
	repo = &Repository{Features: []string{"x"}, changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::minor", repo.DetermineChangeLabel(priority5, map[string]string{"feat": "change::minor"}, "change::standard"))
	repo = &Repository{changeLabels: map[string]struct{}{}}
	assert.Equal(t, "change::standard", repo.DetermineChangeLabel(priority5, nil, "change::standard"))

	// < 2 priority → disabled
	assert.Equal(t, "", repo.DetermineChangeLabel([]string{"change::major"}, nil, ""))
	assert.Equal(t, "", repo.DetermineChangeLabel(nil, nil, ""))
}
