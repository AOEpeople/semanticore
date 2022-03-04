package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithub(t *testing.T) {
	testmux := http.NewServeMux()
	testserver := httptest.NewServer(testmux)
	defer testserver.Close()

	github := NewGithubBackend("test-token", "my/testrepo")

	github.server = testserver.URL
	assert.Error(t, github.request(http.MethodGet, "notfound", http.StatusAccepted, nil, nil))
	assert.NoError(t, github.request(http.MethodGet, "notfound", http.StatusNotFound, nil, nil))

	testmux.HandleFunc("/repos/my/testrepo/testbody", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"foo": "bar"}`)
	})
	var body struct {
		Foo string `json:"foo"`
	}
	assert.NoError(t, github.request(http.MethodGet, "/testbody", http.StatusOK, nil, &body))
	assert.Equal(t, "bar", body.Foo)

	testmux.HandleFunc("/repos/my/testrepo/brokenbody", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `-invalidjson-`)
	})
	assert.Error(t, github.request(http.MethodGet, "/brokenbody", http.StatusOK, nil, &body))

	noMrs := true
	testmux.HandleFunc("/repos/my/testrepo/pulls", func(w http.ResponseWriter, r *http.Request) {
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
			"number": 3,
			"state": "open",
			"head": {
				"ref": "semanticore/release"
			}
		}]`)
	})

	num, err := github.findOpenMergeRequest()
	assert.ErrorIs(t, err, errNoMergeRequestFound)
	assert.Equal(t, 0, num)

	noMrs = false
	num, err = github.findOpenMergeRequest()
	assert.NoError(t, err)
	assert.Equal(t, 3, num)

	testmux.HandleFunc("/repos/my/testrepo/pulls/3", func(w http.ResponseWriter, r *http.Request) {})
	assert.NoError(t, github.CloseMergeRequest())

	assert.NoError(t, github.MergeRequest("main", "Release v1.2.3", "release description", "tag1,tag2"))
	noMrs = true
	assert.NoError(t, github.MergeRequest("main", "Release v1.2.3", "release description", "tag1,tag2"))

	testmux.HandleFunc("/repos/my/testrepo/releases", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	assert.NoError(t, github.Tag("main", "v1.2.3"))

	testmux.HandleFunc("/repos/my/testrepo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"default_branch": "main"}`)
	})
	branch, err := github.MainBranch()
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)
}
