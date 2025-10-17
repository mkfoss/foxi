//go:build mage

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mkfoss/foxi/internal/styles"
)

// Info displays the available mage commands and their descriptions
func Info() {
	fmt.Println(styles.Header("Mage build script for foxi"))
	fmt.Println()
	fmt.Println(styles.Info("Available commands:"))
	fmt.Println()
	fmt.Println(styles.Info("🧪 Quality Commands:"))
	fmt.Println(styles.Example("ci", "Run full CI pipeline (format, test, lint)"))
	fmt.Println(styles.Example("test", "Run all tests"))
	fmt.Println(styles.Example("lint", "Run golangci-lint on project code"))
	fmt.Println(styles.Example("lintall", "Run golangci-lint on project code and magefiles"))
	fmt.Println(styles.Example("format", "Format Go code using gofmt"))
	fmt.Println()
	fmt.Println(styles.Info("📋 Version & Release:"))
	fmt.Println(styles.Example("version", "Display current version from VERSION file"))
	fmt.Println(styles.Example("release", "Create and push annotated release tag"))
	fmt.Println(styles.Example("publish", "Run checks, build, and publish new release"))
	fmt.Println()
	fmt.Println(styles.Info("🔍 Git Commands:"))
	fmt.Println(styles.Example("git:committed", "Check if git repository has no uncommitted changes"))
	fmt.Println(styles.Example("git:pushed", "Check if all commits are pushed to remote"))
	fmt.Println()
	fmt.Printf("%s %s\n", styles.Info("Usage:"), "mage <command>")
	fmt.Println(styles.Dim("Examples:"))
	fmt.Printf("%s %s\n", styles.Dim("  Run CI pipeline:"), styles.Code("mage ci"))
	fmt.Printf("%s %s\n", styles.Dim("  Check version:"), styles.Code("mage version"))
	fmt.Printf("%s %s\n", styles.Dim("  Create release:"), styles.Code("mage release"))
	fmt.Printf("%s %s\n", styles.Dim("  Full publish:"), styles.Code("mage publish"))
	fmt.Printf("%s %s\n", styles.Dim("Tip:"), "Run 'mage -l' to list all available commands")
	fmt.Println()
	fmt.Println(styles.Success("Ready to go!"))
}

// CI runs the full CI pipeline: format, test, lint
func CI() error {
	fmt.Println(styles.Header("🚀 Running CI pipeline..."))
	fmt.Println()

	// Format code first
	if err := Format(); err != nil {
		return err
	}
	fmt.Println()

	// Run tests
	if err := Test(); err != nil {
		return err
	}
	fmt.Println()

	// Run linting last
	if err := Lint(); err != nil {
		return err
	}
	fmt.Println()

	fmt.Println(styles.Success("🎉 CI pipeline completed successfully!"))
	return nil
}

// Test runs all tests in the project
func Test() error {
	fmt.Println(styles.Info("Running tests..."))

	if err := sh.RunV("go", "test", "./...", "-count=1"); err != nil {
		return fmt.Errorf("%s tests failed: %v", styles.Error("Error:"), err)
	}

	fmt.Println(styles.Success("✓ All tests passed"))
	return nil
}

// Lint runs golangci-lint on the project (excludes magefiles)
func Lint() error {
	fmt.Println(styles.Info("Running golangci-lint..."))

	if err := sh.RunV("golangci-lint", "run"); err != nil {
		return fmt.Errorf("%s linting failed: %v", styles.Error("Error:"), err)
	}

	fmt.Println(styles.Success("✓ Linting completed successfully"))
	return nil
}

// LintAll runs golangci-lint on the project including magefiles
func LintAll() error {
	fmt.Println(styles.Info("Running golangci-lint (including magefiles)..."))

	if err := sh.RunV("golangci-lint", "run", "--build-tags=mage"); err != nil {
		return fmt.Errorf("%s linting failed: %v", styles.Error("Error:"), err)
	}

	fmt.Println(styles.Success("✓ Linting completed successfully"))
	return nil
}

// Format runs gofmt on all Go files in the project
func Format() error {
	fmt.Println(styles.Info("Formatting Go code with gofmt..."))

	if err := sh.RunV("gofmt", "-s", "-w", "."); err != nil {
		return fmt.Errorf("%s formatting failed: %v", styles.Error("Error:"), err)
	}

	fmt.Println(styles.Success("✓ Code formatting completed successfully"))
	return nil
}

// GetVersion reads the version from the VERSION file
func GetVersion() (string, error) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "", fmt.Errorf("failed to read VERSION file: %w", err)
	}

	version := strings.TrimSpace(string(data))
	if version == "" {
		return "", fmt.Errorf("VERSION file is empty")
	}

	// Validate version format (semantic versioning)
	if matched, _ := regexp.MatchString(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?$`, version); !matched {
		return "", fmt.Errorf("invalid version format: %s (expected format: x.y.z or vx.y.z)", version)
	}

	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	return version, nil
}

// Version displays the current version
func Version() error {
	version, err := GetVersion()
	if err != nil {
		return err
	}

	fmt.Printf("%s Current version: %s\n", styles.Info("📋"), styles.Success(version))
	return nil
}

// CheckGitStatus ensures the git repository is in a clean state
func CheckGitStatus() error {
	// Check for uncommitted changes
	output, err := sh.Output("git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if strings.TrimSpace(output) != "" {
		return fmt.Errorf("%s repository has uncommitted changes. Commit or stash changes before publishing", styles.Error("Error:"))
	}

	// Check if we're on main/master branch
	currentBranch, err := sh.Output("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	currentBranch = strings.TrimSpace(currentBranch)
	if currentBranch != "main" && currentBranch != "master" {
		fmt.Printf("%s Warning: Publishing from branch '%s' (not main/master)\n", styles.Warning("⚠️"), currentBranch)
		fmt.Print("Continue? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			return fmt.Errorf("publish cancelled")
		}
	}

	return nil
}

// CheckVersionBump ensures the version has been bumped since the last tag
func CheckVersionBump(version string) error {
	// Get the latest tag
	latestTag, err := sh.Output("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		// No existing tags, this is the first release
		fmt.Println(styles.Info("📋 No existing tags found - this will be the first release"))
		return nil
	}

	latestTag = strings.TrimSpace(latestTag)

	// Check if version already exists as a tag
	if latestTag == version {
		return fmt.Errorf("%s version %s already exists as a git tag. Please bump the version in the VERSION file", styles.Error("Error:"), version)
	}

	fmt.Printf("%s Version bump detected: %s → %s\n", styles.Info("📋"), styles.Dim(latestTag), styles.Success(version))
	return nil
}

// Release creates and pushes a new release tag
func Release() error {
	fmt.Println(styles.Header("🚀 Creating release..."))
	fmt.Println()

	// Get version
	version, err := GetVersion()
	if err != nil {
		return err
	}

	// Check git status
	if err := CheckGitStatus(); err != nil {
		return err
	}

	// Check version bump
	if err := CheckVersionBump(version); err != nil {
		return err
	}

	return createAndPushTag(version)
}

// createAndPushTag creates and pushes a Git tag with annotation
func createAndPushTag(version string) error {
	// Create annotated tag
	fmt.Printf("%s Creating annotated tag %s...\n", styles.Info("🏷️"), styles.Success(version))
	if err := sh.Run("git", "tag", "-a", version, "-m", fmt.Sprintf("Release %s", version)); err != nil {
		return fmt.Errorf("%s failed to create tag: %w", styles.Error("Error:"), err)
	}

	fmt.Printf("%s Pushing tag %s...\n", styles.Info("📤"), styles.Success(version))
	if err := sh.Run("git", "push", "origin", version); err != nil {
		return fmt.Errorf("%s failed to push tag: %w", styles.Error("Error:"), err)
	}

	fmt.Printf("%s Release %s created and pushed successfully!\n", styles.Success("✅"), styles.Success(version))
	return nil
}

// Publish runs all checks, creates a release, and pushes to remote
func Publish() error {
	fmt.Println(styles.Header("🚀 Starting publish process..."))
	fmt.Println()

	// Get version first to display it
	version, err := GetVersion()
	if err != nil {
		return err
	}

	fmt.Printf("%s Publishing version: %s\n", styles.Info("📋"), styles.Success(version))
	fmt.Println()

	// Run all pre-publish checks
	fmt.Println(styles.Info("📋 Running pre-publish checks..."))
	mg.SerialDeps(CheckGitStatus, CI)

	// Check version bump
	if err := CheckVersionBump(version); err != nil {
		return err
	}

	// Push current changes
	fmt.Println(styles.Info("📤 Pushing current branch..."))
	if err := sh.Run("git", "push"); err != nil {
		return fmt.Errorf("%s failed to push current branch: %w", styles.Error("Error:"), err)
	}

	// Create and push tag
	fmt.Printf("%s Creating annotated tag %s...\n", styles.Info("🏷️"), styles.Success(version))
	if err := sh.Run("git", "tag", "-a", version, "-m", fmt.Sprintf("Release %s", version)); err != nil {
		return fmt.Errorf("%s failed to create tag: %w", styles.Error("Error:"), err)
	}

	fmt.Printf("%s Pushing tag %s...\n", styles.Info("📤"), styles.Success(version))
	if err := sh.Run("git", "push", "origin", version); err != nil {
		return fmt.Errorf("%s failed to push tag: %w", styles.Error("Error:"), err)
	}

	fmt.Printf("%s Release %s created and pushed successfully!\n", styles.Success("✅"), styles.Success(version))
	fmt.Println()
	fmt.Printf("%s Successfully published %s!\n", styles.Success("🎉"), styles.Success(version))
	fmt.Printf("%s View release: https://github.com/mkfoss/foxi/releases/tag/%s\n", styles.Info("🔗"), version)
	fmt.Println()

	return nil
}

// Git namespace for git-related commands
type Git mg.Namespace

// Committed checks if the git repository has no uncommitted changes
func (Git) Committed() error {
	if !isGitClean() {
		return fmt.Errorf("%s repository is not clean", styles.Error("Error:"))
	}
	return nil
}

// Pushed checks if all commits have been pushed to the remote repository
func (Git) Pushed() error {
	if !isGitPushed() {
		return fmt.Errorf("%s there are unpushed commits", styles.Error("Error:"))
	}
	return nil
}

// Helper functions for git status checking
func isGitClean() bool {
	output, err := sh.Output("git", "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(output) == ""
}

func isGitPushed() bool {
	// Check if there are unpushed commits
	output, err := sh.Output("git", "log", "--oneline", "@{u}..")
	if err != nil {
		// If we can't get the upstream, assume we need to push
		return false
	}
	return strings.TrimSpace(output) == ""
}

// Default target to run when no target is specified
var Default = Info
