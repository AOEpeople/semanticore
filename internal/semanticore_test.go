package internal

import "testing"

func TestParseCommit(t *testing.T) {
	var cases = []struct {
		commit             string
		typ                CommitType
		scope, description string
		major              bool
	}{
		{`feat(something): test`, TypeFeat, `something`, `test`, false},
		{`bug(something): test`, TypeFix, `something`, `test`, false},
		{`bugfix(something): test`, TypeFix, `something`, `test`, false},
		{`bugfixes(something): test`, TypeFix, `something`, `test`, false},
		{`fix(something): test`, TypeFix, `something`, `test`, false},
		{`fix(something) test`, TypeFix, `something`, `test`, false},
		{`fixes(something) test`, TypeFix, `something`, `test`, false},
		{`feat: test`, TypeFeat, ``, `test`, false},
		{`feat`, TypeOther, ``, `feat`, false},
		{`feat:`, TypeOther, ``, `feat:`, false},
		{`feat:   test   `, TypeFeat, ``, `test`, false},
		{`Feat:   test   `, TypeFeat, ``, `test`, false},
		{`Feat   test   `, TypeFeat, ``, `test`, false},
		{`Feat[ someScope ]   test   `, TypeFeat, `somescope`, `test`, false},
		{`Feat[ someScope ]:   test   `, TypeFeat, `somescope`, `test`, false},
		{`Feature[ someScope ]:   test   `, TypeFeat, `somescope`, `test`, false},
		{`test: test`, TypeTest, ``, `test`, false},
		{`testing: test`, TypeTest, ``, `test`, false},
		{"testing:\n\ttest\n", TypeTest, ``, `test`, false},
		// all possible values
		{`fix(something): test`, TypeFix, `something`, `test`, false},
		{`bug(something): test`, TypeFix, `something`, `test`, false},
		{`feat(something): test`, TypeFeat, `something`, `test`, false},
		{`test(something): test`, TypeTest, `something`, `test`, false},
		{`chore(something): test`, TypeChore, `something`, `test`, false},
		{`update(something): test`, TypeChore, `something`, `test`, false},
		{`ops(something): test`, TypeOps, `something`, `test`, false},
		{`ci(something): test`, TypeOps, `something`, `test`, false},
		{`cd(something): test`, TypeOps, `something`, `test`, false},
		{`build(something): test`, TypeOps, `something`, `test`, false},
		{`doc(something): test`, TypeDocs, `something`, `test`, false},
		{`perf(something): test`, TypePerf, `something`, `test`, false},
		{`refactor(something): test`, TypeRefactor, `something`, `test`, false},
		{`rework(something): test`, TypeRefactor, `something`, `test`, false},
		{`security(something): test`, TypeSecurity, `something`, `test`, false},
		{`sec(something): test`, TypeSecurity, `something`, `test`, false},
		{`invalid(something): test`, TypeOther, ``, `invalid(something): test`, false},
		// major commits
		{"testing:\n\ttest\nBREAKING CHANGE: major commit", TypeTest, ``, `test`, true},
		{"testing!:\n\ttest\n", TypeTest, ``, `test`, true},
		{"testing(scope)!:\n\ttest\n", TypeTest, `scope`, `test`, true},
	}

	for _, c := range cases {
		typ, scope, description, major := ParseCommitMessage(c.commit)
		if typ != c.typ || scope != c.scope || description != c.description || major != c.major {
			t.Errorf("commit %q not parsed: typ: %q != %q, scope: %q != %q, description: %q != %q, major %v != %v", c.commit, c.typ, typ, c.scope, scope, c.description, description, c.major, major)
		}
	}
}

func TestDetectReleaseCommit(t *testing.T) {
	var cases = []struct {
		commit              string
		merge               bool
		major, minor, patch int
	}{
		{"Release v1.2.3", false, 1, 2, 3},
		{"Merge a into b\n\nRelease v1.2.3\n\nFoo bar", true, 1, 2, 3},
		{"multi line\n\nRelease v1.2.3\n\nFoo bar", false, 0, 0, 0},
		{"Release v1.2.3\nfoo", false, 0, 0, 0},
		{"Release v1.2.3\n\nfoo", false, 1, 2, 3},
		{"Fixed Release v1.2.3", false, 0, 0, 0},
		{"Release v1.2.3 was totally broken", false, 0, 0, 0},
		{"Release v1.2.3 (#15)", false, 1, 2, 3},
		{"Release v1.2.3 (#15)", true, 1, 2, 3},
		{"Release v1.2.3 (#15)\n\nCo-authored-by: test", false, 1, 2, 3},
	}
	for _, c := range cases {
		major, minor, patch := DetectReleaseCommit(c.commit, c.merge)
		if major != c.major || minor != c.minor || patch != c.patch {
			t.Errorf("detectReleaseCommit %q failed with %d != %d, %d != %d, %d != %d", c.commit, c.major, major, c.minor, minor, c.patch, patch)
		}
	}
}
