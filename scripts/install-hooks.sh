#!/bin/bash
# Install Juggle hooks for automatic session tracking
# Currently supports: Claude Code

set -e

HOOKS_DIR="$HOME/.claude/hooks"
JUGGLE_BIN="$(which juggle || echo "$HOME/.local/bin/juggle")"

echo "Installing Juggle hooks to $HOOKS_DIR"
echo "(Claude Code integration)"

# Create hooks directory if it doesn't exist
mkdir -p "$HOOKS_DIR"

# Install user-prompt-submit hook (tracks activity)
cat > "$HOOKS_DIR/user-prompt-submit" <<'EOF'
#!/bin/bash
# Juggle: Track activity on each user message

JUGGLE_BIN="${JUGGLE_BIN:-juggle}"

# Silently update activity (errors are ignored)
$JUGGLE_BIN track-activity 2>/dev/null || true
EOF

chmod +x "$HOOKS_DIR/user-prompt-submit"
echo "✓ Installed user-prompt-submit hook"

echo ""
echo "Hooks installed successfully!"
echo ""
echo "To start tracking a session, run:"
echo "  juggle start"
echo ""
echo "To see all sessions:"
echo "  juggle status"
echo ""
echo "To find what needs attention:"
echo "  juggle next"
