package locatr

import "embed"

var staticFiles embed.FS

func ReadStaticFile(filename string) ([]byte, error) {
	return staticFiles.ReadFile(filename)
}
