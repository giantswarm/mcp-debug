package project

import "runtime/debug"

// dev is the default for unset build identifiers. Local `go build`
// invocations without ldflags keep this so `mcp-debug --version` stays
// printable and `mcp-debug self-update` keeps refusing to replace a
// development build.
const dev = "dev"

// devel is the module version runtime/debug reports for a build that carries
// no resolvable VCS tag (no .git, or built outside a module checkout).
const devel = "(devel)"

// Build identifiers. The generated Makefile.gen.go.mk stamps all three via
// `-X` ldflags. The architect-orb `go-build` job that produces the release
// assets overrides `gitSHA` (from `CIRCLE_SHA1`) and `buildTimestamp` (UTC
// build time) but does NOT inject `version`; there the version is derived at
// runtime from the Go build info (see Version), which the toolchain stamps
// from the VCS tag — a clean semver when the build sits exactly on a tag, as
// release builds do since they run on the tagged commit.
var (
	version        = dev
	gitSHA         = dev
	buildTimestamp = "unknown"
)

// Version returns the best human-readable build identifier available, in
// order: an explicitly injected `version` ldflag, the VCS version stamped into
// the Go build info, the injected commit SHA, and finally the placeholder
// "dev".
func Version() string {
	if version != dev && version != "" {
		return version
	}
	if v := buildInfoVersion(); v != "" {
		return v
	}
	if gitSHA != dev {
		return gitSHA
	}
	return dev
}

// buildInfoVersion reads the main module version the Go toolchain embedded from
// version control. It returns "" when no usable version is present — either no
// build info, or the "(devel)" placeholder a tag-less build produces — so
// Version can fall through to the next source.
var buildInfoVersion = func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if v := info.Main.Version; v != "" && v != devel {
		return v
	}
	return ""
}

// GitSHA returns the commit SHA the binary was built from, or "dev" when no
// ldflag was injected.
func GitSHA() string { return gitSHA }

// BuildTimestamp returns the UTC build time in RFC 3339 format, or
// "unknown" when no ldflag was injected.
func BuildTimestamp() string { return buildTimestamp }
