# Adding a VCS Operation

Based on `internal/vcs/vcs.go` interface

## Steps

1. **Add method to VCS interface** in `internal/vcs/vcs.go:31-64`:
   ```go
   type VCS interface {
       // ... existing methods ...

       // MyOperation performs a new VCS operation
       MyOperation(projectDir string) error
   }
   ```

2. **Implement for JJ** in `internal/vcs/jj.go`:
   ```go
   func (j *JJBackend) MyOperation(projectDir string) error {
       cmd := exec.Command("jj", "my-command", "args")
       cmd.Dir = projectDir
       output, err := cmd.CombinedOutput()
       if err != nil {
           return fmt.Errorf("jj my-command failed: %w\n%s", err, output)
       }
       return nil
   }
   ```

3. **Implement for Git** in `internal/vcs/git.go`:
   ```go
   func (g *GitBackend) MyOperation(projectDir string) error {
       cmd := exec.Command("git", "my-command", "args")
       cmd.Dir = projectDir
       output, err := cmd.CombinedOutput()
       if err != nil {
           return fmt.Errorf("git my-command failed: %w\n%s", err, output)
       }
       return nil
   }
   ```

4. **Add tests** in `internal/vcs/vcs_test.go` or `internal/integration_test/vcs_*.go`

## Reference Files
- VCS interface: `internal/vcs/vcs.go:31-64`
- JJ implementation example: `internal/vcs/jj.go:101-108` (DescribeWorkingCopy)
- Git implementation example: `internal/vcs/git.go:118-124` (DescribeWorkingCopy)
