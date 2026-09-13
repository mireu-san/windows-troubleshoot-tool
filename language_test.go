package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguagePreferencePersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	app := NewApp()
	if app.GetLanguageSettings().Preference != "auto" {
		t.Fatal("fresh installation must follow system")
	}
	for _, language := range []string{"en", "ja", "ko", "auto"} {
		got, err := app.SetLanguage(language)
		if err != nil {
			t.Fatal(err)
		}
		if got.Preference != language || NewApp().GetLanguageSettings().Preference != language {
			t.Fatal("preference not persisted", got)
		}
	}
	app.SetLanguage("ja")
	if _, err := app.SetLanguage("../../invalid"); err == nil {
		t.Fatal("invalid preference accepted")
	}
	if NewApp().GetLanguageSettings().Language != "ja" {
		t.Fatal("invalid setting changed saved preference")
	}
	path, _ := languageFile()
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	if NewApp().GetLanguageSettings().Preference != "auto" {
		t.Fatal("corrupt settings should follow system")
	}
}
func TestLanguageSaveFailureKeepsSelection(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	if err := os.WriteFile(filepath.Join(dir, "WindowsSystemRepairHelper"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	if _, err := app.SetLanguage("ja"); err == nil {
		t.Fatal("expected save failure")
	}
	if app.GetLanguageSettings().Preference != "auto" {
		t.Fatal("failed save changed preference")
	}
}
func TestNativeLanguages(t *testing.T) {
	for _, language := range []string{"en", "ja", "ko"} {
		if resolveLanguage(language, "other") != language {
			t.Fatal(language)
		}
		if translateNative("Windows 시스템 복구 도우미", language) == "" {
			t.Fatal(language)
		}
	}
	if resolveLanguage("auto", "de") != "en" {
		t.Fatal("unsupported system language must use English")
	}
	if resolveLanguage("auto", "ja") != "ja" {
		t.Fatal("Japanese system language")
	}
	if translateNative("알 수 없는 원문", "en") != "알 수 없는 원문" {
		t.Fatal("unknown text must be preserved")
	}
}
