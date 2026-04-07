<div align="center">
    <img src="vanity-logo.png" alt="Vanity Logo">
    <h1>Vanity</h1>
    <p>An opinionated, minimalist static site generator for redirecting vanity Go module URLs. Take control of your Go modules by owning your namespace.</p>
    <a href="https://github.com/treyburn/vanity/actions"><img src="https://github.com/treyburn/vanity/actions/workflows/ci.yaml/badge.svg" alt="GHA Build"></a>
    <a href="https://codecov.io/gh/treyburn/vanity"><img src="https://codecov.io/gh/treyburn/vanity/graph/badge.svg?token=15ANARDU98" alt="codecov"></a>
</div>

---

## Why?
Go is unique. Unlike other languages with centralized package registries (npm, PyPI, Crates.io, etc.), Go embraced a fully distributed module system. Your domain is your namespace. Yet GitHub has become the de facto registry, bringing vendor lock-in with it. Vanity fixes that.

### Ownership
GitHub doesn't own your code; you do.

Owning your Go module namespace makes you sovereign over your code. This sets you free to switch to any git provider behind the scenes with minimal disruption to your users. No more vendor lock-in.

### Simplicity
Vanity generates plain-Jane static HTML pages. No servers, no databases, no runtime dependencies, no BS. Point to your domain, push your pages, and you're done.

Vanity makes decisions so you don't have to. With the built-in templates, most users won't need to configure anything but their domain and list of modules.

---

## Installation

You can install vanity directly with your Go toolchain.

```shell
go install go.treyburn.dev/vanity@latest
```

Once installed, you can bootstrap your local config with the required fields.

```shell
vanity init
```

---

## Usage
Vanity is driven by a single YAML file and a handful of CLI commands. See [CLI Commands](#cli-commands) for available commands, [Configuration](#configuration) for setting up your `.vanity.yml`, and [Customization](#customization) for tweaking the output.

For most users, it is as simple to get up and running as running `vanity init`, fulling in a few required values, then running `vanity generate`. With that you're ready to ship your static pages.

### CLI Commands
```yaml
Usage: vanity <command> [flags]
 A static site generator for Go vanity URLs.

Flags:
  -h, --help                 Show context-sensitive help.
      --log-level=STRING     Override log level (debug, info, warn, error) ($VANITY_LOG_LEVEL).
      --log-format=STRING    Override log format (text, json) ($VANITY_LOG_FORMAT).

Commands:
  init [flags]
    Generate a minimal .vanity.yml config file (use --verbose for full spec).

  check [flags]
    Validate the .vanity.yml configuration.

  generate [flags]
    Generate static HTML files from configuration.

  preview [flags]
    Generate in-memory and serve via local HTTP.

  serve [flags]
    Serve already-generated output directory.

  clean [flags]
    Remove the generated output directory.

  version [flags]
    Print version information.

Help:
  Run "vanity <command> --help" for more information on a command.
```

### Configuration
All configuration is handled via the `.vanity.yml` YAML file. You can generate a minimal configuration file with just the required fields by running:
```shell
vanity init
```

Or you can generate a full configuration on all possible fields with comments to the purpose by running:
```shell
vanity init --verbose
```

See the [Configuration Reference](#configuration-reference) for the full set of all available configurations.

#### Required fields
The following fields are required to be populated.

```yaml
# .vanity.yml

# Your vanity URL domain (may include subdomains and url paths)
domain: example.com # REQUIRED
# Module definitions (at least one required)
modules:
  - name: my-module # REQUIRED: import path becomes {domain}/{module.name}
    repo: https://github.com/example/my-module # REQUIRED: full git repository URL
```

### Customization
Coming soon.

Bring-your-own-templates is not supported yet.

---

## Examples
As you might have guessed, Vanity uses Vanity to provide it's vanity URL redirection.

I've set this up to push to Cloudflare Pages for my `go.treyburn.dev` domain as an action on publish. You can check out the [repository for that here](https://github.com/treyburn), and I have a small [write-up on my blog](https://treyburn.dev).

---

## Configuration Reference
```yaml
# .vanity.yml

# CLI behavior (overridable via --flags and VANITY_* env vars)
log:
  level: info # Options: debug | info | warn | error
  format: text # Options: text (human-friendly) | json (structured)
  color: true # Colorize text output (no effect on json)
# Output settings
output:
  dir: dist # Relative to .vanity.yml location
  clean: true # Remove output dir before generating
  index: true # Generate root index.html listing all modules
  not_found: true # Generate 404.html redirecting to index
  robots: true # Generate robots.txt (permissive, links to sitemap)
  sitemap: true # Generate sitemap.xml listing all module URLs
# Your vanity URL domain (may include subdomains and url paths)
domain: example.com # REQUIRED
# Default values applied to all modules (overridable per-module)
defaults:
  branch: main # Used in go-source meta tag URL templates (defaults to 'main')
  go_source: true # Include go-source meta tag for pkg.go.dev source links
  redirect_root: https://pkg.go.dev # Redirect = redirect_root/domain/name
# Module definitions (at least one required)
modules:
  - name: my-module # REQUIRED: import path becomes {domain}/{module.name}
    repo: https://github.com/example/my-module # REQUIRED: full git repository URL
    branch: main # Override defaults.branch for this module
    go_source: true # Override defaults.go_source for this module
    redirect: https://pkg.go.dev/example.com/my-module # Override browser redirect URL for this module
    local_path: ./my-module # Local checkout path (default: in-memory clone from repo remote if not specified)
    # Subpackage discovery settings
    subpackages:
      mode: auto # Options: off | auto | explicit (defaults to 'auto')
      exclude: # Directories to skip in auto mode (defaults to [internal, testdata] in 'auto' mode)
        - internal
        - testdata
      paths: # Allow-list exact subpackage paths (explicit mode only)
        - sub/pkg
```
