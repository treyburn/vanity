// Package tmpl provides the shared template infrastructure for vanity.
//
// It owns the embedded base HTML/text templates, the Template interface
// abstraction, and the FuncMap of helper functions available to user
// templates. Both the generator (for rendering) and config (for
// validation parse-checks) depend on this package, keeping the template
// contract in a single place.
package tmpl
