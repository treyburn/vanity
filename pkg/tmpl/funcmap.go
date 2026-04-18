package tmpl

import (
	"fmt"
	"strings"
	"time"

	html "html/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// FuncMap is the set of template functions available to user templates.
// It is shared between the generator (for rendering) and config validation
// (for parse-checking user templates).
var FuncMap = html.FuncMap{
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"title":     cases.Title(language.English).String,
	"join":      strings.Join,
	"sprintf":   fmt.Sprintf,
	"now":       func() time.Time { return time.Now().UTC() },
	"year":      func() int { return time.Now().Year() },
	"contains":  strings.Contains,
	"hasPrefix": strings.HasPrefix,
	"hasSuffix": strings.HasSuffix,
	"replace":   strings.ReplaceAll,
	"trimSpace": strings.TrimSpace,
}
