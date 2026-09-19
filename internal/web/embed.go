package web

import "embed"

//go:embed *.html *.css *.js lib/*.js
var FS embed.FS
