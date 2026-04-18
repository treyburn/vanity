# Custom Templates Example

A minimal example showing how to style vanity's generated pages with custom HTML templates and CSS.

## What's here

```
.vanity.yml              # config with templates + assets
css/style.css            # simple stylesheet
templates/
  index.html             # custom index page (head + body blocks)
  header.html            # reusable nav partial
  footer.html            # reusable footer partial
```

Only the index page is customized. Module, submodule, and 404 pages use vanity's built-in defaults.

## Running it

```shell
# preview the example
cd examples/custom-templates
vanity preview
```

Then open http://localhost:8080 in your browser.

## What it demonstrates

- `{{define "head"}}` -- injecting a `<link>` stylesheet tag into the `<head>`
- `{{define "body"}}` -- replacing the default body with a styled module listing
- `{{template "header" .}}` / `{{template "footer" .}}` -- composing reusable partials
- `{{year}}` and `{{$.Domain}}` -- using FuncMap helpers and template context
- `assets: ["css/"]` -- copying static files into the output directory
