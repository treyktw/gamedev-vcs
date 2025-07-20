#!/bin/bash

echo "🔧 Installing VCS CLI to user PATH..."

# Get the current directory (project root)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
echo "📁 Project root: $PROJECT_ROOT"

# Build the CLI
echo "🔨 Building VCS CLI..."
go build -o bin/vcs ./cmd/vcs

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Build successful!"

# Create user's local bin directory if it doesn't exist
USER_BIN="$HOME/.local/bin"
mkdir -p "$USER_BIN"

# Create a symlink in user's local bin
echo "🔗 Creating symlink in $USER_BIN..."
ln -sf "$PROJECT_ROOT/bin/vcs" "$USER_BIN/vcs"

if [ $? -ne 0 ]; then
    echo "❌ Failed to create symlink!"
    exit 1
fi

# Make it executable
chmod +x "$USER_BIN/vcs"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$USER_BIN:"* ]]; then
    echo "📝 Adding $USER_BIN to PATH..."
    
    # Detect shell
    if [[ "$SHELL" == *"zsh"* ]]; then
        SHELL_RC="$HOME/.zshrc"
    elif [[ "$SHELL" == *"bash"* ]]; then
        SHELL_RC="$HOME/.bashrc"
    else
        SHELL_RC="$HOME/.profile"
    fi
    
    echo "" >> "$SHELL_RC"
    echo "# VCS CLI PATH" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$USER_BIN\"" >> "$SHELL_RC"
    
    echo "✅ Added to $SHELL_RC"
    echo "🔄 Please restart your terminal or run: source $SHELL_RC"
fi

echo "✅ VCS CLI installed successfully!"
echo "🚀 You can now run 'vcs' from anywhere in your terminal"
echo "📖 Try: vcs --help"
echo ""
echo "💡 If 'vcs' command is not found, restart your terminal or run:"
echo "   source $SHELL_RC" 