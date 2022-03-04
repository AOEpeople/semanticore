package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitlab(t *testing.T) {
	testmux := http.NewServeMux()
	testserver := httptest.NewServer(testmux)
	defer testserver.Close()

	gitlab := NewGitlabBackend("test-token", "server", "my/test/repo")

	gitlab.server = testserver.URL
	assert.Error(t, gitlab.request(http.MethodGet, "notfound", http.StatusAccepted, nil, nil))
	assert.NoError(t, gitlab.request(http.MethodGet, "notfound", http.StatusNotFound, nil, nil))

	testmux.HandleFunc("/api/v4/testbody", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"foo": "bar"}`)
	})
	var body struct {
		Foo string `json:"foo"`
	}
	assert.NoError(t, gitlab.request(http.MethodGet, "testbody", http.StatusOK, nil, &body))
	assert.Equal(t, "bar", body.Foo)

	testmux.HandleFunc("/api/v4/brokenbody", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `-invalidjson-`)
	})
	assert.Error(t, gitlab.request(http.MethodGet, "brokenbody", http.StatusOK, nil, &body))

	noMrs := true
	testmux.HandleFunc("/api/v4/projects/my/test/repo/merge_requests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		if noMrs {
			fmt.Fprint(w, `[]`)
			return
		}
		fmt.Fprint(w, `[{
			"id": 123,
			"iid": 3,
			"source_branch": "semanticore/release",
			"state": "opened"
		}]`)
	})

	num, err := gitlab.findOpenMergeRequest()
	assert.ErrorIs(t, err, errNoMergeRequestFound)
	assert.Equal(t, 0, num)

	noMrs = false
	num, err = gitlab.findOpenMergeRequest()
	assert.NoError(t, err)
	assert.Equal(t, 3, num)

	testmux.HandleFunc("/api/v4/projects/my/test/repo/merge_requests/3", func(w http.ResponseWriter, r *http.Request) {})
	assert.NoError(t, gitlab.CloseMergeRequest())

	assert.NoError(t, gitlab.MergeRequest("main", "Release v1.2.3", "release description", "tag1,tag2"))
	noMrs = true
	assert.NoError(t, gitlab.MergeRequest("main", "Release v1.2.3", "release description", "tag1,tag2"))

	testmux.HandleFunc("/api/v4/projects/my/test/repo/repository/tags", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	assert.NoError(t, gitlab.Tag("v1.2.3", "abc123"))

	testmux.HandleFunc("/api/v4/projects/my/test/repo/releases", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	assert.NoError(t, gitlab.Release("v1.2.3"))

	testmux.HandleFunc("/api/v4/projects/my/test/repo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"default_branch": "main"}`)
	})
	branch, err := gitlab.MainBranch()
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)
}
