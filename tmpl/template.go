package tmpl

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

var funcMap = template.FuncMap{
	"get": func(m map[string]any, key string) (any, error) {
		v, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("key %q not found in map", key)
		}
		return v, nil
	},
	"default": func(val any, fallback string) string {
		if val == nil {
			return fallback
		}
		s := fmt.Sprintf("%v", val)
		if s == "" {
			return fallback
		}
		return s
	},
}

var templateCache sync.Map // map[string]*template.Template

func Eval(tmpl string, data map[string]any) (string, error) {
	t, err := getOrParseTemplate(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

func getOrParseTemplate(tmpl string) (*template.Template, error) {
	if cached, ok := templateCache.Load(tmpl); ok {
		return cached.(*template.Template), nil
	}

	t, err := template.New("eval").
		Option("missingkey=error").
		Funcs(funcMap).
		Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	templateCache.Store(tmpl, t)
	return t, nil
}
