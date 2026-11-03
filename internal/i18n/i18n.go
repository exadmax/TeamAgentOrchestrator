// Package i18n holds every message the user can see, in English (the default) and Brazilian Portuguese.
//
// Catalogs live in locales/<lang>/*.json (one file per area; all files of a language are merged).
// Values are fmt formats: T("tasks.not_found", id) with "task %s does not exist". Use explicit indexes
// (%[2]s) when a translation needs the arguments in another order.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"sync/atomic"
)

const (
	EN = "en"
	PT = "pt-BR"
)

// Languages lists the supported languages, default first.
var Languages = []string{EN, PT}

//go:embed locales
var files embed.FS

var (
	catalogs = load()
	current  atomic.Value // string
)

func load() map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, lang := range Languages {
		out[lang] = map[string]string{}
		names, _ := fs.Glob(files, path.Join("locales", lang, "*.json"))
		for _, name := range names {
			b, err := files.ReadFile(name)
			if err != nil {
				panic(err)
			}
			part := map[string]string{}
			if err := json.Unmarshal(b, &part); err != nil {
				panic(fmt.Sprintf("i18n: %s: %v", name, err))
			}
			for k, v := range part {
				if _, dup := out[lang][k]; dup {
					panic(fmt.Sprintf("i18n: key %q repeated in %s", k, name))
				}
				out[lang][k] = v
			}
		}
	}
	return out
}

// Supported reports whether lang has a catalog.
func Supported(lang string) bool {
	_, ok := catalogs[lang]
	return ok
}

// Set changes the language of every message from now on. An unknown language is an error.
func Set(lang string) error {
	if !Supported(lang) {
		return fmt.Errorf("unsupported language %q", lang)
	}
	current.Store(lang)
	return nil
}

// Lang returns the current language.
func Lang() string {
	if l, ok := current.Load().(string); ok {
		return l
	}
	return EN
}

// T translates key into the current language.
func T(key string, args ...any) string { return TL(Lang(), key, args...) }

// TL translates key into lang. A key missing in lang falls back to English, then to the key itself.
func TL(lang, key string, args ...any) string {
	format, ok := catalogs[lang][key]
	if !ok {
		if format, ok = catalogs[EN][key]; !ok {
			format = key
		}
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// Err is T as an error; %w in the format wraps an error argument.
func Err(key string, args ...any) error {
	format, ok := catalogs[Lang()][key]
	if !ok {
		if format, ok = catalogs[EN][key]; !ok {
			format = key
		}
	}
	return fmt.Errorf(format, args...)
}

// Keys returns the sorted keys of lang (for tests and tools).
func Keys(lang string) []string {
	out := make([]string, 0, len(catalogs[lang]))
	for k := range catalogs[lang] {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Format returns the raw format of key in lang ("" if missing).
func Format(lang, key string) string { return catalogs[lang][key] }
