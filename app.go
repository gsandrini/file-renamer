package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx  context.Context
	lang string
}

var translations = map[string]map[string]string{
	"it": {
		"scanError":   "Errore durante la scansione: ",
		"renameError": "Errore rinominando ",
		"noChanges":   "Nessuna modifica necessaria.",
		"success":     "Rinominati %d elementi con successo.",
	},
	"en": {
		"scanError":   "Error during scan: ",
		"renameError": "Error renaming ",
		"noChanges":   "No changes needed.",
		"success":     "Renamed %d items successfully.",
	},
}

// RenameEntry represents a single file rename operation
type RenameEntry struct {
	OldPath     string `json:"oldPath"`
	NewPath     string `json:"newPath"`
	OldRelative string `json:"oldRelative"`
	NewRelative string `json:"newRelative"`
	IsDir       bool   `json:"isDir"`
}

// RenameResult is the result of a rename operation
type RenameResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{lang: "it"}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SetLanguage sets the active language for backend messages
func (a *App) SetLanguage(lang string) {
	if _, ok := translations[lang]; ok {
		a.lang = lang
	}
}

// t is a helper that returns the translated string for a key
func (a *App) t(key string) string {
	if msgs, ok := translations[a.lang]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	return key
}

// SelectDirectory opens a native directory picker dialog
func (a *App) SelectDirectory() string {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Seleziona directory",
	})
	if err != nil || dir == "" {
		return ""
	}
	return dir
}

// Formatting helpers
var camelSplit = regexp.MustCompile(`([a-z])([A-Z])`)

// normalize splits a name into lowercase words, stripping separators
func normalizeWords(name string) []string {
	// Replace common separators with space, then split
	name = camelSplit.ReplaceAllString(name, "$1 $2")
	r := strings.NewReplacer("-", " ", "_", " ")
	return strings.Fields(strings.ToLower(r.Replace(name)))
}

// formatDirname formats a directory segment according to the chosen style
func formatDirname(name, style string) string {
	words := normalizeWords(name)
	if len(words) == 0 {
		return name
	}
	switch style {
	case "snake": // Directory_Name
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
		return strings.Join(words, "_")
	case "camel": // DirectoryName
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
		return strings.Join(words, "")
	default: // kebab → Directory-Name
		for i, w := range words {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
		return strings.Join(words, "-")
	}
}

// formatFilename formats a file segment according to the chosen style
func formatFilename(name, style string) string {
	ext := strings.ToLower(filepath.Ext(name))
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	words := normalizeWords(stem)
	if len(words) == 0 {
		return name
	}
	var newStem string
	switch style {
	case "snake": // file_name
		newStem = strings.ToLower(strings.Join(words, "_"))
	case "camel": // fileName
		for i, w := range words {
			if i == 0 {
				words[i] = strings.ToLower(w)
			} else {
				words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
			}
		}
		newStem = strings.Join(words, "")
	default: // kebab → file-name
		newStem = strings.ToLower(strings.Join(words, "-"))
	}
	if ext != "" {
		return newStem + ext
	}
	return newStem
}

// Core logic

// PreviewRenames returns the list of rename operations without applying them
func (a *App) PreviewRenames(root string, style string) ([]RenameEntry, error) {
	var entries []RenameEntry
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, de := range dirEntries {
		path := filepath.Join(root, de.Name())
		entry, changed, e := buildEntry(root, path, de.IsDir(), style)
		if e != nil {
			return nil, e
		}
		if changed {
			entries = append(entries, entry)
		}
	}

	// Sort: directories first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].OldRelative < entries[j].OldRelative
	})

	return entries, nil
}

func buildEntry(root, path string, isDir bool, style string) (RenameEntry, bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return RenameEntry{}, false, err
	}

	parts := strings.Split(rel, string(filepath.Separator))
	newParts := make([]string, len(parts))
	for i, part := range parts {
		isLast := i == len(parts)-1
		if isLast && !isDir {
			newParts[i] = formatFilename(part, style)
		} else {
			newParts[i] = formatDirname(part, style)
		}
	}

	newRel := strings.Join(newParts, string(filepath.Separator))
	newPath := filepath.Join(root, newRel)

	changed := newPath != path
	return RenameEntry{
		OldPath:     path,
		NewPath:     newPath,
		OldRelative: rel,
		NewRelative: newRel,
		IsDir:       isDir,
	}, changed, nil
}

// DirEntry represents a single entry in a directory listing
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

// ListDirectory returns the top-level entries of a directory (unsorted raw names)
func (a *App) ListDirectory(root string) ([]DirEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var result []DirEntry
	for _, e := range entries {
		result = append(result, DirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	// Sort: dirs first, then alphabetically
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// ApplyRenames performs the actual rename operations
func (a *App) ApplyRenames(root string, style string) RenameResult {
	entries, err := a.PreviewRenames(root, style)
	if err != nil {
		return RenameResult{Success: false, Message: a.t("scanError") + err.Error()}
	}

	if len(entries) == 0 {
		return RenameResult{Success: true, Message: a.t("noChanges"), Count: 0}
	}

	// Process in reverse order so children are renamed before parents
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if err := os.Rename(e.OldPath, e.NewPath); err != nil {
			return RenameResult{
				Success: false,
				Message: a.t("renameError") + e.OldRelative + ": " + err.Error(),
			}
		}
	}

	return RenameResult{
		Success: true,
		Message: fmt.Sprintf(a.t("success"), len(entries)),
		Count:   len(entries),
	}
}
