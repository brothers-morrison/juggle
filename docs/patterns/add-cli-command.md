# Adding a New CLI Command

Based on `internal/cli/show.go`

## Steps

1. **Create new file** `internal/cli/mycommand.go`:
   ```go
   package cli

   import (
       "github.com/ohare93/juggle/internal/session"
       "github.com/spf13/cobra"
   )

   var myCmd = &cobra.Command{
       Use:   "my <args>",
       Short: "Brief description",
       Long:  `Longer description...`,
       Args:  cobra.ExactArgs(1), // or other validator
       RunE:  runMy,
   }

   func init() {
       // Add flags if needed
       myCmd.Flags().BoolVar(&myFlag, "flag", false, "description")
   }

   func runMy(cmd *cobra.Command, args []string) error {
       // Get working directory
       cwd, err := GetWorkingDir()
       if err != nil {
           return fmt.Errorf("failed to get current directory: %w", err)
       }

       // Create store for accessing balls
       store, err := NewStoreForCommand(cwd)
       if err != nil {
           return fmt.Errorf("failed to initialize store: %w", err)
       }

       // Your command logic here
       return nil
   }
   ```

2. **Register in `internal/cli/root.go`** init() function:
   ```go
   rootCmd.AddCommand(myCmd)
   ```

3. **Add integration test** in `internal/integration_test/`

## Reference Files
- Example: `internal/cli/show.go` (simple read command)
- Registration: `internal/cli/root.go:206-223`
