package webfs

import (
	"embed"
	"fmt"
)

//go:embed pages/*.html

var pages embed.FS

func Page(name string) ([]byte, error) {

	return pages.ReadFile(fmt.Sprintf("pages/%s", name))
}

func Status(status int) ([]byte, error) {
	name := fmt.Sprintf("%d.html", status)
	return Page(name)
}
