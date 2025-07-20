#!/bin/bash

echo "🔧 Installing VCS CLI to system PATH..."

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

# Create a symlink in /usr/local/bin (requires sudo)
echo "🔗 Creating symlink in /usr/local/bin..."
sudo ln -sf "$PROJECT_ROOT/bin/vcs" /usr/local/bin/vcs

if [ $? -ne 0 ]; then
    echo "❌ Failed to create symlink!"
    echo "💡 Try running this script with sudo: sudo ./scripts/install-cli.sh"
    exit 1
fi

# Make it executable
sudo chmod +x /usr/local/bin/vcs

echo "✅ VCS CLI installed successfully!"
echo "🚀 You can now run 'vcs' from anywhere in your terminal"
echo "�� Try: vcs --help" 