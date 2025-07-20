package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	serverURL string
	verbose   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "vcs",
		Short:   "Game Development Version Control System",
		Long:    `A modern version control system built specifically for game development. Handles binary assets, real-time collaboration, and UE5 integration. July 17, 2025 Updates and fixes!`,
		Version: version,
	}

	// ───── Global Flags ─────────────────────────────────────────────
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "VCS server URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// ───── Core VCS Commands ────────────────────────────────────────
	rootCmd.AddCommand(initCmd())   // Initialize a new repo
	rootCmd.AddCommand(cloneCmd())  // Clone an existing repo
	rootCmd.AddCommand(addCmd())    // Add files
	rootCmd.AddCommand(commitCmd()) // Commit changes
	rootCmd.AddCommand(pushCmd())   // Push to remote
	rootCmd.AddCommand(pullCmd())   // Pull from remote
	rootCmd.AddCommand(statusCmd()) // View current status

	// ───── Locking / Asset Collaboration ────────────────────────────
	rootCmd.AddCommand(lockCmd())   // Lock a file or asset
	rootCmd.AddCommand(unlockCmd()) // Unlock a file or asset

	// ───── Branching and Versioning ────────────────────────────────
	rootCmd.AddCommand(branchCmd())  // Manage branches
	rootCmd.AddCommand(migrateCmd()) // Migrate project schema
	rootCmd.AddCommand(cleanCmd())   // Clean unused branches or data

	// ───── Project Lifecycle / Initialization ──────────────────────
	rootCmd.AddCommand(initVCSCmd()) // `vcs init` command for new projects

	// ───── Real-time & Watcher Tools ───────────────────────────────
	rootCmd.AddCommand(watchCmd())   // Watch for file changes
	rootCmd.AddCommand(storageCmd()) // View/manage storage usage
	rootCmd.AddCommand(cleanupCmd()) // Cleanup temp or orphaned files

	// ───── Analytics and Insights ──────────────────────────────────
	rootCmd.AddCommand(analyticsCmd()) // View commit and usage analytics

	// ───── Authentication & User Management ────────────────────────
	rootCmd.AddCommand(loginCmd())   // Log in (supports --google)
	rootCmd.AddCommand(signupCmd())  // Quick signup
	rootCmd.AddCommand(logoutCmd())  // Logout
	rootCmd.AddCommand(whoamiCmd())  // Show logged-in user info
	rootCmd.AddCommand(accountCmd()) // Account management

	// ───── Utility / Debugging Commands ────────────────────────────
	rootCmd.AddCommand(testCmd()) // Test server connectivity

	// ───── Execute Root Command ────────────────────────────────────
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func testCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test connection to VCS server",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("🧪 Testing VCS server connection...\n")
			fmt.Printf("Server URL: %s\n", serverURL)

			// Test basic connectivity
			if err := testServerConnectivity(); err != nil {
				return err
			}

			// Test API endpoints
			client, _ := NewAPIClient(serverURL)

			// Test health endpoint
			_, err := client.makeRequest("GET", "/health", nil)
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			fmt.Printf("✅ Health endpoint working\n")

			// Test ready endpoint
			_, err = client.makeRequest("GET", "/ready", nil)
			if err != nil {
				return fmt.Errorf("ready check failed: %w", err)
			}
			fmt.Printf("✅ Ready endpoint working\n")

			// Test auth endpoints
			_, err = client.makeRequest("GET", "/api/v1/auth/google", nil)
			if err == nil || !strings.Contains(err.Error(), "request failed with status 500") {
				fmt.Printf("✅ Google OAuth endpoint available\n")
			} else {
				fmt.Printf("⚠️  Google OAuth may need configuration\n")
			}

			fmt.Printf("🎉 Basic tests passed! Server is working.\n")
			fmt.Printf("\n💡 Try these commands:\n")
			fmt.Printf("  vcs signup                    # Create new account\n")
			fmt.Printf("  vcs login                     # Username/password login\n")
			fmt.Printf("  vcs login --google            # Google OAuth login\n")

			return nil
		},
	}
}
