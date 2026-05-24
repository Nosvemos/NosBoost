package config

import (
	"embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	currentLang   = "en"
	langMutex     sync.RWMutex
	translations  = make(map[string]map[string]string)
	languageNames = make(map[string]string)
)

func init() {
	// Parse all embedded JSON files from the locales directory
	files, err := localesFS.ReadDir("locales")
	if err != nil {
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		langCode := strings.TrimSuffix(file.Name(), ".json")
		data, err := localesFS.ReadFile("locales/" + file.Name())
		if err != nil {
			continue
		}

		var dict map[string]string
		if err := json.Unmarshal(data, &dict); err == nil {
			translations[langCode] = dict
			if name, ok := dict["lang_name"]; ok {
				languageNames[langCode] = name
			} else {
				languageNames[langCode] = strings.ToUpper(langCode)
			}
		}
	}
}

// SetLanguage sets the active language dynamically
func SetLanguage(lang string) {
	langMutex.Lock()
	defer langMutex.Unlock()
	if _, exists := translations[lang]; exists {
		currentLang = lang
	}
}

// GetLanguage retrieves the current language code
func GetLanguage() string {
	langMutex.RLock()
	defer langMutex.RUnlock()
	return currentLang
}

// GetAvailableLanguages returns the registered language codes and their display names
func GetAvailableLanguages() map[string]string {
	langMutex.RLock()
	defer langMutex.RUnlock()
	res := make(map[string]string)
	for k, v := range languageNames {
		res[k] = v
	}
	return res
}

// T translates a key using the active language dictionary
func T(key string) string {
	langMutex.RLock()
	defer langMutex.RUnlock()
	if dict, exists := translations[currentLang]; exists {
		if val, ok := dict[key]; ok {
			return val
		}
	}
	// Fallback to English
	if dict, exists := translations["en"]; exists {
		if val, ok := dict[key]; ok {
			return val
		}
	}
	return key
}
