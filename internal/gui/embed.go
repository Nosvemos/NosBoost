package gui

import _ "embed"

// appIconPNG holds the embedded application icon (PNG format).
// Fyne accepts PNG bytes directly via fyne.NewStaticResource.
//
//go:embed appicon.png
var appIconPNG []byte
