# Adding a New Ball Field

Based on commit `7ed3384` which added `AgentProvider` and `ModelOverride`

## Steps

1. **Add field to Ball struct** in `internal/session/ball.go:76-97`:
   ```go
   type Ball struct {
       // ... existing fields ...
       MyNewField string `json:"my_new_field,omitempty"`
   }
   ```

2. **Add validation function** (if needed) in `internal/session/ball.go`:
   ```go
   func ValidateMyNewField(s string) bool {
       // Validation logic
       return true
   }
   ```

3. **Add setter method** in `internal/session/ball.go`:
   ```go
   func (b *Ball) SetMyNewField(value string) {
       b.MyNewField = value
       b.UpdateActivity()  // Updates LastActivity and UpdateCount
   }
   ```

4. **Update CLI** in `internal/cli/update.go` to allow setting the field via `juggle update`

5. **Update TUI** in:
   - `internal/tui/ball_form.go` - Add form field for editing
   - `internal/tui/view.go` - Display field in ball view
   - `internal/tui/split_handlers.go` - Add keyboard handlers if needed

6. **Update export** in `internal/cli/export.go` if field should be included in exports

## Reference Files
- Ball struct: `internal/session/ball.go:76-97`
- Example setters: `internal/session/ball.go:553-593` (SetModelSize, SetAgentProvider, SetModelOverride)
- CLI update: `internal/cli/update.go`
- TUI form: `internal/tui/ball_form.go`
