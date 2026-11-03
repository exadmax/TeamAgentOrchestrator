package i18n

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var verb = regexp.MustCompile(`%(\[\d+\])?[-+# 0]*\d*(\.\d+)?[vsdqwxXfgtT]`)

// verbs returns the fmt verbs of a format, sorted, ignoring explicit indexes so a reordered
// translation still matches ("%s ... %d" == "%[2]d ... %[1]s").
func verbs(format string) []string {
	var out []string
	for _, v := range verb.FindAllString(format, -1) {
		out = append(out, v[len(v)-1:])
	}
	sort.Strings(out)
	return out
}

func TestCatalogsMatch(t *testing.T) {
	en, pt := Keys(EN), Keys(PT)
	if len(en) == 0 {
		t.Fatal("English catalog is empty")
	}
	missing := func(a, b []string) []string {
		set := map[string]bool{}
		for _, k := range b {
			set[k] = true
		}
		var out []string
		for _, k := range a {
			if !set[k] {
				out = append(out, k)
			}
		}
		return out
	}
	if m := missing(en, pt); len(m) > 0 {
		t.Errorf("missing in pt-BR: %v", m)
	}
	if m := missing(pt, en); len(m) > 0 {
		t.Errorf("missing in en: %v", m)
	}
	for _, k := range en {
		if a, b := strings.Join(verbs(Format(EN, k)), ","), strings.Join(verbs(Format(PT, k)), ","); a != b {
			t.Errorf("%s: placeholders differ: en %q, pt-BR %q", k, a, b)
		}
		if strings.TrimSpace(Format(PT, k)) == "" || strings.TrimSpace(Format(EN, k)) == "" {
			t.Errorf("%s: empty translation", k)
		}
	}
}

// The frontend catalogs use {name} placeholders; same keys and same names in both languages.
func TestFrontendCatalogsMatch(t *testing.T) {
	dir := filepath.Join("..", "..", "frontend", "src", "i18n")
	read := func(lang string) map[string]string {
		b, err := os.ReadFile(filepath.Join(dir, lang+".json"))
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	en, pt := read(EN), read(PT)
	if len(en) == 0 {
		t.Fatal("frontend English catalog is empty")
	}
	names := regexp.MustCompile(`\{(\w+)\}`)
	params := func(s string) string {
		var out []string
		for _, m := range names.FindAllStringSubmatch(s, -1) {
			out = append(out, m[1])
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	for k, v := range en {
		p, ok := pt[k]
		if !ok {
			t.Errorf("frontend: %s missing in pt-BR", k)
			continue
		}
		if params(v) != params(p) {
			t.Errorf("frontend: %s: placeholders differ: en %q, pt-BR %q", k, params(v), params(p))
		}
	}
	for k := range pt {
		if _, ok := en[k]; !ok {
			t.Errorf("frontend: %s missing in en", k)
		}
	}
}

func TestTranslateAndFallback(t *testing.T) {
	defer Set(EN)
	if Lang() != EN {
		t.Fatalf("default language = %s", Lang())
	}
	if err := Set("xx"); err == nil {
		t.Fatal("unknown language accepted")
	}
	if got := T("no.such.key"); got != "no.such.key" {
		t.Fatalf("missing key = %q", got)
	}
	k := Keys(EN)[0]
	_ = Set(PT)
	if T(k) != Format(PT, k) && len(verbs(Format(PT, k))) == 0 {
		t.Fatalf("T(%s) in pt-BR = %q", k, T(k))
	}
	base := errors.New("boom")
	if err := Err("i18n.wrapped", base); !errors.Is(err, base) {
		t.Fatalf("Err does not wrap: %v", err)
	}
}
