package config

// AppVersion represents the active application release version.
// It can be overridden at build time using compiler flags:
// -ldflags="-X 'nosboost/internal/config.AppVersion=v1.0.0'"
var AppVersion = "v1.0.0"
