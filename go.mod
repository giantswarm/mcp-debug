module github.com/giantswarm/mcp-debug

go 1.26.0

require (
	github.com/chzyer/readline v1.5.1
	github.com/creativeprojects/go-selfupdate v1.6.0
	github.com/mark3labs/mcp-go v0.58.0
	github.com/spf13/cobra v1.10.2
)

require (
	code.gitea.io/sdk/gitea v0.23.2 // indirect
	github.com/42wim/httpsig v1.2.4 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/davidmz/go-pageant v1.0.2 // indirect
	github.com/go-fed/httpsig v1.1.0 // indirect
	github.com/google/go-github/v86 v86.0.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	gitlab.com/gitlab-org/api/client-go v1.46.0 // indirect
	golang.org/x/crypto v0.56.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Pin a transitive module flagged by the OSS Index scan (nancy) in CI: nothing
// here imports golang.org/x/mod, so go mod tidy resolves it to the version the
// dependency graph asks for (v0.38.0, CVE-2026-56864/56865); the replace holds
// the fixed release.
replace golang.org/x/mod => golang.org/x/mod v0.40.0
