package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
)

//go:embed frontend/src/locales/messages.json
var messageCatalog []byte
var translations = func() map[string]map[string]string {
	var catalog map[string]map[string]string
	if err := json.Unmarshal(messageCatalog, &catalog); err != nil {
		panic(err)
	}
	return catalog
}()

type LanguageSettings struct {
	Preference string `json:"preference"`
	Language   string `json:"language"`
}

func validLanguage(value string) bool {
	return value == "auto" || value == "ko" || value == "en" || value == "ja"
}
func resolveLanguage(preference, system string) string {
	if preference != "auto" && validLanguage(preference) {
		return preference
	}
	if system == "ko" || system == "ja" {
		return system
	}
	return "en"
}
func languageFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "WindowsSystemRepairHelper", "language.json"), nil
}
func readLanguagePreference() string {
	path, err := languageFile()
	if err != nil {
		return "auto"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "auto"
	}
	var saved struct {
		Preference string `json:"preference"`
	}
	if json.Unmarshal(data, &saved) != nil || !validLanguage(saved.Preference) {
		return "auto"
	}
	return saved.Preference
}
func translateNative(source, language string) string {
	if translated := translations[source][language]; translated != "" {
		return translated
	}
	return source
}
func nativeText(source string) string {
	return translateNative(source, resolveLanguage(readLanguagePreference(), systemLanguage()))
}
func (a *App) GetLanguageSettings() LanguageSettings {
	a.languageMu.RLock()
	defer a.languageMu.RUnlock()
	preference := a.languagePreference
	if !validLanguage(preference) {
		preference = "auto"
	}
	return LanguageSettings{preference, resolveLanguage(preference, systemLanguage())}
}
func (a *App) SetLanguage(preference string) (LanguageSettings, error) {
	if !validLanguage(preference) {
		return a.GetLanguageSettings(), errors.New("Unsupported language")
	}
	path, err := languageFile()
	if err != nil {
		return a.GetLanguageSettings(), err
	}
	a.languageMu.Lock()
	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err == nil {
		data, _ := json.Marshal(struct {
			Preference string `json:"preference"`
		}{preference})
		err = os.WriteFile(path, data, 0600)
	}
	if err == nil {
		a.languagePreference = preference
	}
	a.languageMu.Unlock()
	settings := a.GetLanguageSettings()
	if err == nil && a.ctx != nil {
		runtime.WindowSetTitle(a.ctx, translateNative("Windows 시스템 복구 도우미", settings.Language))
	}
	return settings, err
}
