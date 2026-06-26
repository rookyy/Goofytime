package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"stundenerfassung/models"
)

var Templates *template.Template

func InitTemplates() {
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"formatHours": func(h float64) string {
			hours := int(h)
			minutes := int((h - float64(hours)) * 60 + 0.5)
			if minutes == 60 {
				hours++
				minutes = 0
			}
			if minutes > 0 {
				return fmt.Sprintf("%dh %dm", hours, minutes)
			}
			return fmt.Sprintf("%dh", hours)
		},
		"upper": strings.ToUpper,
		"slice": func(s string, a, b int) string {
			if a < 0 || b > len(s) || a > b {
				return s
			}
			return s[a:b]
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"headerTitle": func() string {
			return models.GetAdminSettings().HeaderTitle
		},
		"footerText": func() string {
			return models.GetAdminSettings().FooterText
		},
	}

	Templates = template.Must(template.New("").Funcs(funcMap).ParseGlob(
		filepath.Join("templates", "*.html"),
	))
}

func RenderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := Templates.ExecuteTemplate(w, name, data); err != nil {
		fmt.Printf("Template error (%s): %v\n", name, err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
