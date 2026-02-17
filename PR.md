## Summary

I was having trouble remembering the set of commands (session, add folders, add balls, agent run?) to get juggle going on a new project.  I love the idea of juggle and it's filling a need that I had to be able to better see, understand, and prioritize work/tasks/balls that I want to run with agents (thanks for building this!).  Further, I'm trying to get the most out of my AI subscriptions, so I'm attempting to seek out and understand how to run and manage agent loops in parallel.  This seemed like an excellent tool for that, but with a steep learning curve. (I still have not got 2x OpenCode sessions at the same time, should I be using nohup bg ?)
So this PR attempts to lessen the learning curve, making it easier for a beginner to get started especially for the 2nd or 3rd time when he (me) can't remember the correct command order, or what all is required for getting juggle working.

## Changes

1. Adds command line arg --help-quickstart (colorized markdown with lipgloss), which shows the same quickstart section from the readme, but this time on the command line, so that I don't have to leave my vm/cloud machine to go look it up again. 

2. Refactor quickstart text into it's own separate .md file, so that it can be included into readme.md (same as before, no changes) but have that same text also shown as a response to command arg --help-quickstart
by placing it in it's own file, the hope is that it can be edited in one place only (for future) single source of truth (SSOT) and still be up-to-date in both other places (readme.md and command line arg content).

3. third part is adding to the "juggle import spec" functionality to look for a spec.md or PRD.md file and convert that into balls.
it uses ## headers to divide the markdown file up into separate tasks/balls.  Just trying to save time and make this more interoperable so that I don't have to transpose a bunch of stuff by hand if I've already got a Spec.md file written.

## Testing

- [x] `devbox run test-quiet` passes
- [x] Manual testing performed
- [x] `go test ./internal/specparser/...` — 20 test functions covering H2 parsing, priority/model-size tags, checkbox/bullet/numbered lists, directory scanning, case-insensitive file detection, realistic PRD input, and edge cases (empty files, no criteria, no context, missing files)
- [x] `juggle --help-quickstart` prints styled quickstart guide with ANSI colors (headers, code blocks, blockquotes, inline bold/code)
- [x] `juggle --help` shows new `--help-quickstart` flag and quickstart URL
- [x] `juggle import spec --dry-run` on a sample spec.md correctly previews parsed balls with title, priority, model size, acceptance criteria, and source file
- [x] `juggle import spec` creates balls, skips duplicates on re-run, and tags with `spec:<filename>`
- [x] `scripts/sync-quickstart.sh` syncs quickstart.md content into README between markers without corrupting surrounding content

## Checklist

- [x] Code follows existing patterns in the codebase
- [x] Documentation updated (if applicable)
  - README.md updated with auto-sync markers for quickstart section
  - `juggle --help` updated to include `--help-quickstart` flag and quickstart URL
- [x] Tests added for new functionality (if applicable)
  - 757-line test file (`internal/specparser/specparser_test.go`) with 20 test functions for the spec parser
- [x] New `internal/specparser` package follows existing package conventions (exported types, `_test.go` in same package)
- [x] Uses existing dependencies only (lipgloss for terminal rendering, cobra for CLI)
- [x] New CLI commands registered via `init()` consistent with existing command registration pattern
- [x] Embedded quickstart content uses `go:embed` consistent with Go best practices for static assets
