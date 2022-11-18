package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Gitlab struct {
	server string
	token  string
	repo   string
}

var _ Backend = Gitlab{}

func NewGitlabBackend(token, server, repo string) Gitlab {
	return Gitlab{
		server: "https://" + server,
		token:  token,
		repo:   repo,
	}
}

func (gitlab Gitlab) request(method, endpoint string, expectedStatus int, body io.Reader, target interface{}) error {
	log.Printf("[gitlab] %s: %s", method, gitlab.server+"/api/v4/"+endpoint)
	req, err := http.NewRequest(method, gitlab.server+"/api/v4/"+endpoint, body)
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("PRIVATE-TOKEN", gitlab.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		b, _ := io.ReadAll(resp.Body)
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

var errNoMergeRequestFound = errors.New("no merge request found")

func (gitlab Gitlab) findOpenMergeRequest() (int, error) {
	var mrs []struct {
		ID           int    `json:"id"`
		IID          int    `json:"iid"`
		SourceBranch string `json:"source_branch"`
		State        string `json:"state"`
	}

	if err := gitlab.request(http.MethodGet, fmt.Sprintf("projects/%s/merge_requests?state=opened&source_branch=semanticore%%2frelease", url.PathEscape(gitlab.repo)), http.StatusOK, nil, &mrs); err != nil {
		return 0, fmt.Errorf("unable to get merge requests: %w", err)
	}

	for _, mr := range mrs {
		if mr.SourceBranch == "semanticore/release" && mr.State == "opened" {
			log.Printf("[gitlab] merge request found: %d", mr.IID)
			return mr.IID, nil
		}
	}

	return 0, errNoMergeRequestFound
}

func (gitlab Gitlab) CloseMergeRequest() error {
	iid, err := gitlab.findOpenMergeRequest()
	if errors.Is(err, errNoMergeRequestFound) {
		return nil
	}
	if err != nil {
		return err
	}

	data := make(url.Values)
	data.Set("state_event", "close")
	return gitlab.request(http.MethodPut, fmt.Sprintf("projects/%s/merge_requests/%d", url.PathEscape(gitlab.repo), iid), http.StatusOK, strings.NewReader(data.Encode()), nil)
}

func (gitlab Gitlab) MergeRequest(target, title, description, labels string) error {
	iid, err := gitlab.findOpenMergeRequest()
	if err != nil && !errors.Is(err, errNoMergeRequestFound) {
		return err
	}

	data := make(url.Values)
	data.Set("source_branch", "semanticore/release")
	data.Set("target_branch", target)
	data.Set("title", title)
	data.Set("description", description)
	data.Set("squash", "true")
	data.Set("remove_source_branch", "true")
	data.Set("labels", labels)

	if iid > 0 {
		return gitlab.request(http.MethodPut, fmt.Sprintf("projects/%s/merge_requests/%d", url.PathEscape(gitlab.repo), iid), http.StatusOK, strings.NewReader(data.Encode()), nil)
	}
	return gitlab.request(http.MethodPost, fmt.Sprintf("projects/%s/merge_requests", url.PathEscape(gitlab.repo)), http.StatusCreated, strings.NewReader(data.Encode()), nil)
}

func (gitlab Gitlab) Release(tag, ref, changelog string) error {
	data := make(url.Values)
	data.Set("tag_name", tag)
	data.Set("ref", ref)
	if err := gitlab.request(http.MethodPost, fmt.Sprintf("projects/%s/repository/tags", url.PathEscape(gitlab.repo)), http.StatusCreated, strings.NewReader(data.Encode()), nil); err != nil {
		return fmt.Errorf("unable to tag release %s on %s: %w", tag, ref, err)
	}

	data = make(url.Values)
	data.Set("tag_name", tag)
	data.Set("description", changelog)
	return gitlab.request(http.MethodPost, fmt.Sprintf("projects/%s/releases", url.PathEscape(gitlab.repo)), http.StatusCreated, strings.NewReader(data.Encode()), nil)
}

func (gitlab Gitlab) MainBranch() (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}

	if err := gitlab.request(http.MethodGet, fmt.Sprintf("projects/%s", url.PathEscape(gitlab.repo)), http.StatusOK, nil, &repo); err != nil {
		return "", fmt.Errorf("unable to get repository: %w", err)
	}

	return repo.DefaultBranch, nil
}

func (gitlab Gitlab) SetAuth(r *http.Request) {
	r.SetBasicAuth("gitlab-ci-token", gitlab.token)
}

func (gitlab Gitlab) Name() string {
	return "gitlab-auth"
}

func (gitlab Gitlab) String() string {
	masked := "*******"
	if gitlab.token == "" {
		masked = "<empty>"
	}

	return fmt.Sprintf("%s - %s", gitlab.Name(), masked)
}
