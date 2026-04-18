package generator

import (
	"fmt"
	"strings"
	"time"

	html "html/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// funcMap returns the template functions available to user templates.
var funcMap = html.FuncMap{
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"title":     cases.Title(language.English).String,
	"join":      strings.Join,
	"sprintf":   fmt.Sprintf,
	"now":       func() time.Time { return time.Now().UTC().Truncate(24 * time.Hour) },
	"year":      func() int { return time.Now().Year() },
	"contains":  strings.Contains,
	"hasPrefix": strings.HasPrefix,
	"hasSuffix": strings.HasSuffix,
	"replace":   strings.ReplaceAll,
	"trimSpace": strings.TrimSpace,
}
