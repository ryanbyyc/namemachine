package namemachine

import "embed"

//go:embed lists/*/*.txt
var listsFS embed.FS
