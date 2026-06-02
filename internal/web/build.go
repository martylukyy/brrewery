package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var Dist embed.FS

func DistFS() (fs.FS, error) {
	return fs.Sub(Dist, "dist")
}
