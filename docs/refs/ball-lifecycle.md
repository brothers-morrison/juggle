# Ball Lifecycle File References

- **Ball struct definition**: `internal/session/ball.go:76-97`
- **State transitions**: `internal/session/ball.go:262-280` (Start, Complete, Block methods)
- **JSONL persistence**: `internal/session/store.go:100-150` (AppendBall, UpdateBall)
- **Archival**: `internal/session/archive.go:20-80`
