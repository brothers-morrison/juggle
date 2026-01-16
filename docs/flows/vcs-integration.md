# VCS Integration Flow

1. On ball activation: `vcs.GetCurrentRevision()` stores in `ball.StartingRevision`
2. Optional: `vcs.DescribeWorkingCopy()` updates VCS description with ball info
3. On ball complete/block: `vcs.GetCurrentRevision()` stores in `ball.RevisionID`
4. VCS backend determined by: project config → global config → auto-detect (.jj/ or .git/)

## Key Files
- VCS interface: `internal/vcs/vcs.go:31-64`
- Detection logic: `internal/vcs/detect.go:20-50`
- JJ backend: `internal/vcs/jj.go`
- Git backend: `internal/vcs/git.go`
