package bumper

import (
	"os"
	"os/exec"
	"strings"

	t "github.com/sjc5/kit/pkg/cliutil"
)

func Run() {
	// Confirm user has pushed code
	t.Blue("have you pushed your code? ")
	t.RequireYes("aborted. go commit and push your changes, then come back")

	// Get current tag
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		t.Plain("No existing tags found. Get started by running:")
		t.NewLine()

		t.Plain("```sh")
		t.NewLine()

		t.Blue("git tag v0.0.1")
		t.NewLine()

		t.Blue("git push origin v0.0.1")
		t.NewLine()

		t.Blue("GOPROXY=proxy.golang.org go list -m all")
		t.NewLine()

		t.Plain("```")
		t.NewLine()

		t.Exit("Aborted", nil)
	}

	// Clean current tag
	currentTagStr := strings.TrimSpace(string(output))
	if currentTagStr == "" {
		t.Exit("current tag is empty", nil)
	}

	// Show current tag
	t.Plain("current tag: ")
	t.Green(currentTagStr)
	t.NewLine()

	// Ask for new version
	t.Blue("what is the new version? ")
	t.Plain("v")
	version, err := t.NewReader().ReadString('\n')
	if err != nil {
		t.Exit("failed to read version", err)
	}

	trimmedVersion := strings.TrimSpace(version)
	if trimmedVersion == "" {
		t.Exit("version is empty", nil)
	}

	bumpedVersion := "v" + trimmedVersion

	// Show new tag
	t.Plain("Result: ")
	t.Red(currentTagStr)
	t.Plain("  -->  ")
	t.Green(bumpedVersion)
	t.NewLine()

	// Ask for confirmation
	t.Blue("is this correct? ")
	t.RequireYes("aborted")

	// Ask for git push confirmation
	t.Blue("apply tag ")
	t.Green(bumpedVersion)
	t.Blue(" and push to git? ")
	t.RequireYes("aborted")

	// Create new tag
	t.Plain("creating new tag")
	cmd = exec.Command("git", "tag", bumpedVersion)
	t.TryCmd(cmd, "tag creation failed")

	// Push new tag
	t.Plain("pushing new tag")
	cmd = exec.Command("git", "push", "origin", bumpedVersion)
	t.TryCmd(cmd, "tag push failed")

	// Update go proxy
	t.Plain("updating go proxy")
	cmd = exec.Command("go", "list", "-m", "all")
	cmd.Env = append(os.Environ(), "GOPROXY=proxy.golang.org")
	t.TryCmd(cmd, "go proxy update failed")

	// Done
	t.Plain("done")
}
