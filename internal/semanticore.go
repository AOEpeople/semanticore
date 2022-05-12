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
}
