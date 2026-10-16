package brew

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bjet007/brewupdater/internal/plan"
)

const (
	binaryName = "brew"
	// upgradeWaitDelay is how long an interrupted upgrade gets to clean up before it is killed.
	upgradeWaitDelay = 30 * time.Second
)

type runFunc func(cmd *exec.Cmd) error

// Client runs the brew binary on PATH.
type Client struct {
	run runFunc
}

// New returns a Client backed by the brew binary on PATH.
func New() *Client {
	return &Client{run: (*exec.Cmd).Run}
}

// Update refreshes tap metadata.
func (c *Client) Update(ctx context.Context) error {
	_, err := c.capture(ctx, UpdateArgs())
	return err
}

// Outdated lists outdated formulae and casks. greedy includes casks that update themselves.
func (c *Client) Outdated(ctx context.Context, greedy bool) ([]plan.Package, error) {
	out, err := c.capture(ctx, OutdatedArgs(greedy))
	if err != nil {
		return nil, err
	}
	return ParseOutdated(out)
}

// Installed lists the names of every installed formula and cask.
func (c *Client) Installed(ctx context.Context) ([]string, error) {
	out, err := c.capture(ctx, InstalledArgs())
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(out)), nil
}

// Upgrade hands the terminal to brew so its output streams live and sudo prompts work.
// Cancelling ctx interrupts brew rather than killing it, so it does not leave a half-installed package.
// An empty names list does nothing.
func (c *Client) Upgrade(ctx context.Context, kind plan.Kind, names []string, greedy bool) error {
	if len(names) == 0 {
		return nil
	}

	args := UpgradeArgs(kind, names, greedy)
	cmd := exec.CommandContext(ctx, binaryName, args...)
	cmd.Env = append(os.Environ(), "HOMEBREW_NO_AUTO_UPDATE=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = upgradeWaitDelay

	if err := c.run(cmd); err != nil {
		return fmt.Errorf("brew %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func (c *Client) capture(ctx context.Context, args []string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binaryName, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := c.run(cmd); err != nil {
		err = fmt.Errorf("brew %s: %w", strings.Join(args, " "), err)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}
