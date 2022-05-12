package internal

import (
	"regexp"
	"strconv"
	"strings"
)

type CommitType string

const (
	TypeFix      CommitType = "fix"
	TypeFeat     CommitType = "feat"
	TypeTest     CommitType = "test"
	TypeChore    CommitType = "chore"
	TypeOps      CommitType = "ops"
	TypeDocs     CommitType = "docs"
	TypePerf     CommitType = "perf"
	TypeRefactor CommitType = "refactor"
	TypeSecurity CommitType = "security"
	TypeOther    CommitType = "other"
)

var commitRegexp = regexp.MustCompile(`#?\d*\s*\[?([a-zA-Z]*)\]?\s*([\(\[]([^\]\)]*)[\]\)])?\s*?(!?)(:?)\s*(.*)`)
var specialChars = strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")

func ParseCommitMessage(msg string) (CommitType, string, string, bool) {
	match := commitRegexp.FindStringSubmatch(msg)
	var commitType, scope, description string
	var typ CommitType
	var major = false

	if len(match) == 7 {
		commitType, scope, description = strings.ToLower(match[1]), strings.ToLower(match[3]), strings.TrimSpace(match[6])
		if match[4] == "!" {
			major = true
		}
		// if we do not have a `:` after type and category we might have a non-conventional commit
		if match[5] != ":" {
			description = msg
		}
	}
	if len(description) == 0 {
		commitType = ""
	}

	if strings.HasPrefix(commitType, "fix") || strings.HasPrefix(commitType, "bug") {
		typ = TypeFix
	} else if strings.HasPrefix(commitType, "feat") {
		typ = TypeFeat
	} else if strings.HasPrefix(commitType, "test") {
		typ = TypeTest
	} else if strings.HasPrefix(commitType, "chore") || strings.HasPrefix(commitType, "update") {
		typ = TypeChore
	} else if strings.HasPrefix(commitType, "ops") || strings.HasPrefix(commitType, "ci") || strings.HasPrefix(commitType, "cd") || strings.HasPrefix(commitType, "build") {
		typ = TypeOps
	} else if strings.HasPrefix(commitType, "doc") {
		typ = TypeDocs
	} else if strings.HasPrefix(commitType, "perf") {
		typ = TypePerf
	} else if strings.HasPrefix(commitType, "refactor") || strings.HasPrefix(commitType, "rework") {
		typ = TypeRefactor
	} else if strings.HasPrefix(commitType, "sec") {
		typ = TypeSecurity
	} else {
		typ = TypeOther
		scope = ""
		description = msg
	}

	scope = strings.TrimSpace(scope)
	commitDescription := ""
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		if len(line) > 0 && commitDescription == "" {
			commitDescription = line
			break
		}
	}
	for _, line := range strings.Split(msg, "\n") {
		if strings.HasPrefix(line, "BREAKING CHANGE:") {
			major = true
			break
		}
	}

	scope = specialChars.Replace(scope)
	commitDescription = specialChars.Replace(commitDescription)

	return typ, scope, commitDescription, major
}

var releaseCommitRegex = regexp.MustCompile(`^Release (v?)(\d+).(\d+).(\d+)( \(.*\))?$`)

func DetectReleaseCommit(commit string, merge bool) (vPrefix string, major, minor, patch int) {
	candidates := []string{strings.SplitN(commit, "\n\n", 2)[0]}
	if merge {
		candidates = strings.Split(commit, "\n")
	}
	for _, candidate := range candidates {
		matches := releaseCommitRegex.FindStringSubmatch(candidate)
		if matches != nil {
			vPrefix = matches[1]
			major, _ = strconv.Atoi(matches[2])
			minor, _ = strconv.Atoi(matches[3])
			patch, _ = strconv.Atoi(matches[4])
			return
		}
	}
	return "v", 0, 0, 0
}
