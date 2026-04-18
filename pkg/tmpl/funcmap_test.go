package tmpl

import (
	"bytes"
	"testing"
	"time"

	html "html/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncMap_TemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     any
		expected string
	}{
		{"upper", `{{upper .}}`, "hello", "HELLO"},
		{"lower", `{{lower .}}`, "HELLO", "hello"},
		{"title", `{{title .}}`, "hello world", "Hello World"},
		{"join", `{{join .Elems .Sep}}`, struct {
			Elems []string
			Sep   string
		}{[]string{"a", "b", "c"}, ", "}, "a, b, c"},
		{"sprintf", `{{sprintf "%s/%s" "go.dev" "vanity"}}`, nil, "go.dev/vanity"},
		{"contains true", `{{contains . "bar"}}`, "foobar", "true"},
		{"contains false", `{{contains . "baz"}}`, "foobar", "false"},
		{"hasPrefix true", `{{hasPrefix . "foo"}}`, "foobar", "true"},
		{"hasPrefix false", `{{hasPrefix . "bar"}}`, "foobar", "false"},
		{"hasSuffix true", `{{hasSuffix . "bar"}}`, "foobar", "true"},
		{"hasSuffix false", `{{hasSuffix . "foo"}}`, "foobar", "false"},
		{"replace", `{{replace . "-" "_"}}`, "my-module", "my_module"},
		{"trimSpace", `{{trimSpace .}}`, "  hello  ", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := html.New("test").Funcs(FuncMap).Parse(tt.template)
			require.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestFuncMap_Now(t *testing.T) {
	tmpl, err := html.New("test").Funcs(FuncMap).Parse(`{{now.Format "2006-01-02"}}`)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	expected := time.Now().UTC().Format("2006-01-02")
	assert.Equal(t, expected, buf.String())
}

func TestFuncMap_NowIsUTC(t *testing.T) {
	tmpl, err := html.New("test").Funcs(FuncMap).Parse(`{{now.Format "MST"}}`)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	assert.Equal(t, "UTC", buf.String())
}

func TestFuncMap_Year(t *testing.T) {
	tmpl, err := html.New("test").Funcs(FuncMap).Parse(`{{year}}`)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	require.NoError(t, err)

	expected := time.Now().UTC().Format("2006")
	assert.Equal(t, expected, buf.String())
}
