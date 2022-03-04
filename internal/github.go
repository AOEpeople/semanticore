package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type Github struct {
	server string
	token  string
	repo   string
}

var _ Backend = Github{}

func NewGithubBackend(token, repo string) Github {
	return Github{
		server: "https://api.github.com",
		token:  token,
		repo:   repo,
	}
}

func (github Github) request(method, endpoint string, expectedStatus int, body interface{}, target interface{}) error {
	var bodyReader io.Reader = nil
	if body != nil {
		bodybytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(bodybytes)
	}

	log.Printf("[Github] %s: %s", method, github.server+"/repos/"+github.repo+endpoint)
	req, err := http.NewRequest(method, github.server+"/repos/"+github.repo+endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Authorization", "Bearer "+github.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("expected status is %d: %v %s", expectedStatus, resp, string(b))
	}
	if target == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("unable to decode body: %w", err)
	}
	return nil
}

func (github Github) findOpenMergeRequest() (int, error) {
	var mrs []struct {
		ID    int    `json:"id"`
		IID   int    `json:"number"`
		State string `json:"state"`
		Head  struct {
			Ref string `json:"ref"`
		} `json:"head"`
	}

	if err := github.request(http.MethodGet, "/pulls", http.StatusOK, nil, &mrs); err != nil {
		return 0, fmt.Errorf("unable to get merge requests: %w", err)
	}

	for _, mr := range mrs {
		if mr.Head.Ref == "semanticore/release" && mr.State == "open" {
			log.Printf("[Github] merge request found: %d", mr.IID)
			return mr.IID, nil
		}
	}

	return 0, errNoMergeRequestFound
}

type githubPullBody struct {
	State string `json:"state,omitempty"`
	Base  string `json:"base,omitempty"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Head  string `json:"head,omitempty"`
}

func (github Github) CloseMergeRequest() error {
	iid, err := github.findOpenMergeRequest()
	if errors.Is(err, errNoMergeRequestFound) {
		return nil
	}
	if err != nil {
		return err
	}

	data := githubPullBody{
		State: "closed",
	}
	return github.request(http.MethodPut, fmt.Sprintf("/pulls/%d", iid), http.StatusOK, data, nil)
}

func (github Github) MergeRequest(target, title, description, labels string) error {
	iid, err := github.findOpenMergeRequest()
	if err != nil && !errors.Is(err, errNoMergeRequestFound) {
		return err
	}

	data := githubPullBody{
		Base:  target,
		Title: title,
		Body:  description,
	}
	if iid > 0 {
		return github.request(http.MethodPatch, fmt.Sprintf("/pulls/%d", iid), http.StatusOK, data, nil)
	}
	data.Head = "semanticore/release"
	return github.request(http.MethodPost, "/pulls", http.StatusCreated, data, nil)
}

type githubReleaseBody struct {
	TagName              string `json:"tag_name"`
	TargetCommitish      string `json:"target_commitish"`
	Name                 string `json:"name"`
	GenerateReleaseNotes bool   `json:"generate_release_notes"`
}

func (github Github) Tag(tag, ref string) error {
	data := githubReleaseBody{
		TagName:              tag,
		TargetCommitish:      ref,
		Name:                 tag,
		GenerateReleaseNotes: true,
	}
	return github.request(http.MethodPost, "/releases", http.StatusCreated, data, nil)
}

func (github Github) Release(tag string) error {
	return nil // noop
}

func (github Github) MainBranch() (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}

	if err := github.request(http.MethodGet, "", http.StatusOK, nil, &repo); err != nil {
		return "", fmt.Errorf("unable to get repository: %w", err)
	}

	return repo.DefaultBranch, nil
}

func (github Github) SetAuth(r *http.Request) {
	r.SetBasicAuth("Github-ci-token", github.token)
}

func (github Github) Name() string {
	return "Github-auth"
}

func (github Github) String() string {
	masked := "*******"
	if github.token == "" {
		masked = "<empty>"
	}

	return fmt.Sprintf("%s - %s", github.Name(), masked)
}
