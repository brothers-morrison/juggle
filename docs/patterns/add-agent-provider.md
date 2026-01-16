# Adding a New Agent Provider

Based on `internal/agent/provider/opencode.go`

## Steps

1. **Create provider file** `internal/agent/provider/myprovider.go`:
   ```go
   package provider

   import "os/exec"

   type MyProvider struct{}

   func NewMyProvider() *MyProvider {
       return &MyProvider{}
   }

   func (p *MyProvider) Type() Type {
       return Type("myprovider")
   }

   func (p *MyProvider) Name() string {
       return "My Provider"
   }

   func (p *MyProvider) Run(opts RunOptions) (*RunResult, error) {
       // Build command with provider-specific flags
       args := []string{"run", "--prompt", opts.Prompt}

       if opts.Model != "" {
           args = append(args, "--model", opts.Model)
       }

       cmd := exec.Command("myprovider", args...)
       cmd.Dir = opts.WorkingDir

       // Execute and capture output
       output, err := cmd.CombinedOutput()

       result := &RunResult{
           Output:   string(output),
           ExitCode: cmd.ProcessState.ExitCode(),
       }

       // Parse signals from output (COMPLETE, BLOCKED, etc.)
       parsePromiseSignals(result)

       return result, err
   }
   ```

2. **Add Type constant** in `internal/agent/provider/provider.go:12-17`:
   ```go
   const (
       TypeClaude   Type = "claude"
       TypeOpenCode Type = "opencode"
       TypeMyProvider Type = "myprovider"  // Add this
   )
   ```

3. **Update IsValid()** in `internal/agent/provider/provider.go:24-27`

4. **Add to detection logic** in `internal/agent/provider/detect.go`

5. **Update CLI flag** in `internal/cli/agent.go` to accept new provider name

## Reference Files
- Provider interface: `internal/agent/provider/provider.go:78-94`
- Example implementation: `internal/agent/provider/opencode.go`
- Detection logic: `internal/agent/provider/detect.go`
