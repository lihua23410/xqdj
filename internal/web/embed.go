package web

import "embed"

//go:embed *.html *.css *.js lib/*.js sfx/*.wav
var FS embed.FS
