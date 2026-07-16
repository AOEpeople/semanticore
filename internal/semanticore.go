package internal

import (
	"github.com/go-git/go-git/v5/plumbing/transport"
)

type Backend interface {
	transport.AuthMethod
	Release(tag, ref, changelog string) error
	MergeRequest(target, title, description, labels string) error
	CloseMergeRequest() error
	MainBranch() (string, error)
	// IssuePrefixedLabels returns all labels starting with prefix for the given issue number.
	// Implementations should return nil, nil when issues are not supported or the issue is not found.
	IssuePrefixedLabels(id int, prefix string) ([]string, error)
}
