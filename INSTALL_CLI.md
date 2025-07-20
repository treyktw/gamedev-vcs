# Installing VCS CLI on macOS

There are several ways to install the VCS CLI so you can run `vcs` commands from anywhere in your terminal.

## Option 1: Automatic Installation (Recommended)

### System-wide installation (requires sudo):
```bash
chmod +x scripts/install-cli.sh
./scripts/install-cli.sh
```

### User-only installation (no sudo required):
```bash
chmod +x scripts/install-cli-user.sh
./scripts/install-cli-user.sh
```

## Option 2: Manual Installation

### Step 1: Build the CLI
```bash
go build -o bin/vcs ./cmd/vcs
```

### Step 2: Add to PATH

Choose one of these methods:

#### Method A: System-wide (requires sudo)
```bash
sudo ln -sf "$(pwd)/bin/vcs" /usr/local/bin/vcs
sudo chmod +x /usr/local/bin/vcs
```

#### Method B: User-only
```bash
# Create user bin directory
mkdir -p ~/.local/bin

# Create symlink
ln -sf "$(pwd)/bin/vcs" ~/.local/bin/vcs
chmod +x ~/.local/bin/vcs

# Add to PATH (for zsh)
echo 'export PATH="$PATH:~/.local/bin"' >> ~/.zshrc
source ~/.zshrc

# Or for bash
echo 'export PATH="$PATH:~/.local/bin"' >> ~/.bashrc
source ~/.bashrc
```

## Option 3: Using Go Install

If you want to install it as a Go module:

```bash
# From the project root
go install ./cmd/vcs
```

This will install it to `$GOPATH/bin` or `$GOBIN` if set.

## Verify Installation

After installation, test that it works:

```bash
vcs --help
```

You should see the VCS CLI help output.

## Updating the CLI

When you make changes to the CLI, you'll need to rebuild and reinstall:

```bash
# Rebuild
go build -o bin/vcs ./cmd/vcs

# Reinstall (if using symlinks, they'll automatically point to the new binary)
# If using go install:
go install ./cmd/vcs
```

## Troubleshooting

### "Command not found" error
- Make sure the binary is in your PATH
- Restart your terminal or run `source ~/.zshrc` (or `~/.bashrc`)
- Check that the symlink exists: `ls -la /usr/local/bin/vcs` or `ls -la ~/.local/bin/vcs`

### Permission denied
- Make sure the binary is executable: `chmod +x bin/vcs`
- If using sudo installation, make sure you have sudo privileges

### Build errors
- Make sure you're in the project root directory
- Check that all Go dependencies are installed: `go mod tidy` 