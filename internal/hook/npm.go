package hook

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"regexp"

	"github.com/go-git/go-git/v5"

	"github.com/aoepeople/semanticore/internal"
)

var packagejson string

func init() {
	flag.StringVar(&packagejson, "npm-update-version", "", "enable update of npm package.json version field")
}

func NpmUpdateVersionHook(wt *git.Worktree, repository *internal.Repository) {
	if packagejson == "" {
		return
	}

	f, err := wt.Filesystem.Open(packagejson)
	if err != nil {
		log.Printf("npm-update-version: error opening file %s: %s", packagejson, err)
		return
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		log.Printf("npm-update-version: error reading file: %s", err)
		return
	}
	f.Close()

	var jsonData struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(contents, &jsonData); err != nil {
		log.Printf("npm-update-version: error parsing json: %s", err)
		return
	}
	packagejsonRegexp := regexp.MustCompile(`"version"\s*:\s*"` + jsonData.Version + `"`)
	contents = packagejsonRegexp.ReplaceAll(contents, []byte(fmt.Sprintf(`"version": "%d.%d.%d"`, repository.Major, repository.Minor, repository.Patch)))

	f, err = wt.Filesystem.Create(packagejson)
	if err != nil {
		log.Printf("npm-update-version: error opening file %s for writing: %s", packagejson, err)
		return
	}
	defer f.Close()
	_, err = f.Write(contents)
	if err != nil {
		log.Printf("npm-update-version: error writing file: %s", err)
		return
	}
	wt.Add(packagejson)
}
