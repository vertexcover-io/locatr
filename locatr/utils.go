package locatr

import "embed"

//go:embed meta/htmlMinifier.js meta/locate_element.prompt
var staticFiles embed.FS

func ReadStaticFile(filename string) ([]byte, error) {
	return staticFiles.ReadFile(filename)
}
