package buildinfo

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	Version   = "0.0.0-dev"
	Commit    = ""
	Date      = ""
	UserAgent = ""
)

func init() {
	UserAgent = fmt.Sprintf("brrewery/%s (%s %s)", Version, runtime.GOOS, runtime.GOARCH)
}

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func JSON() ([]byte, error) {
	return json.Marshal(Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	})
}
