package internal

import "strings"

func TrimChangelog(cl []byte, changelogMaxLines int) []byte {
	clLines := strings.Split(string(cl), "\n")

	if len(clLines) < changelogMaxLines {
		return cl
	}

	for i := changelogMaxLines - 1; i > 0; i-- {
		var l = strings.ReplaceAll(clLines[i], " ", "")
		l = strings.ToLower(l)
		if strings.HasPrefix(l, "##version") {
			return []byte(strings.Join(clLines[:i], "\n"))
		}
	}

	return cl
}
