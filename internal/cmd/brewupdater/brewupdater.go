// Package brewupdater wires the config, brew, the checklist and the reporter behind the root command.
package brewupdater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"charm.land/fang/v2"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/bjet007/brewupdater/internal/brew"
	"github.com/bjet007/brewupdater/internal/cli/checklist"
	"github.com/bjet007/brewupdater/internal/cli/reporter"
	"github.com/bjet007/brewupdater/internal/config"
	"github.com/bjet007/brewupdater/internal/plan"
	"github.com/bjet007/brewupdater/internal/upgrade"
)

const cancelledText = "Cancelled"

var errNoTerminal = errors.New("the checklist needs a terminal; use --list to see what is outdated or -y to upgrade the favorites")

type flags struct {
	configPath string
	noUpdate   bool
	greedy     bool
	list       bool
	json       bool
	yes        bool
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	var res upgrade.Result
	a := newApp()
	root := newRootCmd(func(cmd *cobra.Command, f flags) error {
		var err error
		res, err = a.run(cmd.Context(), f)
		return err
	})

	err := fang.Execute(context.Background(), root,
		fang.WithVersion(resolveVersion(version, readBuildInfo())),
		fang.WithErrorHandler(handleError),
		fang.WithNotifySignal(os.Interrupt),
	)
	return exitCode(res, err)
}

func newRootCmd(runE func(*cobra.Command, flags) error) *cobra.Command {
	var f flags
	cmd := &cobra.Command{
		Use:   "brewupdater",
		Short: "Pick the outdated Homebrew packages to upgrade, favorites first",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runE(cmd, f)
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&f.configPath, "config", "", "config file (default $XDG_CONFIG_HOME/brewupdater/config.yaml)")
	fl.BoolVar(&f.noUpdate, "no-update", false, "skip brew update")
	fl.BoolVar(&f.greedy, "greedy", false, "include casks that update themselves")
	fl.BoolVar(&f.list, "list", false, "print the outdated packages and exit")
	fl.BoolVar(&f.json, "json", false, "print the outdated packages as JSON and exit")
	fl.BoolVarP(&f.yes, "yes", "y", false, "skip the checklist and upgrade the favorites")
	cmd.MarkFlagsMutuallyExclusive("list", "json", "yes")

	return cmd
}

// Runner is what a run needs from upgrade.Service.
type Runner interface {
	Plan(ctx context.Context, o upgrade.Options) (plan.Plan, error)
	Run(ctx context.Context, o upgrade.Options) (upgrade.Result, error)
}

type app struct {
	stdout   io.Writer
	stderr   io.Writer
	terminal bool
	// newRunner builds the service with its refresh progress and summary written to progress.
	newRunner func(progress io.Writer) Runner
}

func newApp() app {
	return app{
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		terminal:  isTerminal(os.Stdin) && isTerminal(os.Stdout),
		newRunner: brewRunner,
	}
}

func brewRunner(progress io.Writer) Runner {
	client := brew.New()
	return upgrade.Service{
		Refresher: client,
		Lister:    client,
		Upgrader:  client,
		Selector:  checklist.Prompt{},
		Reporter:  reporter.New(progress),
	}
}

func (a app) run(ctx context.Context, f flags) (upgrade.Result, error) {
	listing := f.list || f.json
	if !listing && !f.yes && !a.terminal {
		return upgrade.Result{}, errNoTerminal
	}

	cfg, err := loadConfig(f.configPath)
	if err != nil {
		return upgrade.Result{}, err
	}
	opts := options(cfg, f)

	if !listing {
		return a.newRunner(a.stdout).Run(ctx, opts)
	}

	// Keep stdout for the list itself, so it can be piped.
	p, err := a.newRunner(a.stderr).Plan(ctx, opts)
	if err != nil {
		return upgrade.Result{}, err
	}
	writeWarnings(a.stderr, p)
	if f.json {
		return upgrade.Result{}, writeJSON(a.stdout, p)
	}
	return upgrade.Result{}, writeList(a.stdout, p)
}

// loadConfig reads the default location when path is empty, where a missing file means defaults.
// A path given with --config must exist.
func loadConfig(path string) (config.Config, error) {
	if path == "" {
		path, err := config.Path()
		if err != nil {
			return config.Config{}, fmt.Errorf("locating config: %w", err)
		}
		return config.Load(path)
	}

	if _, err := os.Stat(path); err != nil {
		return config.Config{}, fmt.Errorf("reading config: %w", err)
	}
	return config.Load(path)
}

// options applies the flags on top of the config. Flags only ever turn a setting on (--greedy) or off (--no-update).
func options(cfg config.Config, f flags) upgrade.Options {
	return upgrade.Options{
		Favorites:  cfg.Favorites,
		Greedy:     cfg.Greedy || f.greedy,
		AutoUpdate: cfg.AutoUpdate && !f.noUpdate,
		Yes:        f.yes,
	}
}

func cancelled(err error) bool {
	return errors.Is(err, checklist.ErrAborted) || errors.Is(err, context.Canceled)
}

func exitCode(res upgrade.Result, err error) int {
	switch {
	case cancelled(err):
		return 0
	case err != nil, !res.OK():
		return 1
	}
	return 0
}

func handleError(w io.Writer, styles fang.Styles, err error) {
	if cancelled(err) {
		_, _ = fmt.Fprintln(w, cancelledText)
		return
	}
	fang.DefaultErrorHandler(w, styles, err)
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(f.Fd())
}
