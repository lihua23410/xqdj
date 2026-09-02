package web

import "embed"

//go:embed *.html *.css *.js faction/*.png
var FS embed.FS
