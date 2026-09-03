package web

import "embed"

//go:embed *.html *.css *.js faction/*.png fx/*.css fx/*.js status/*.png
var FS embed.FS
