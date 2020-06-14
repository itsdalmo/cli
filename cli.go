// Package cli provides the most clean and simple API for building CLI applications (in the authors view).
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/pflag"
)

// ErrMisconfigured is returned when a Command is misconfigured.
type ErrMisconfigured struct {
	cmd *Command
	msg string
}

// Error implements errors.Error.
func (e *ErrMisconfigured) Error() string {
	return fmt.Sprintf("misconfigured command %q: %s", e.cmd.name(), e.msg)
}

// Context ...
type Context struct {
	*pflag.FlagSet
}

// Options ...
type Options struct {
	Reader    io.Reader
	Writer    io.Writer
	ErrWriter io.Writer
	UsageFunc func(*Command) string
}

// complete passes default values to the options that are unset.
func (opts *Options) complete() {
	if opts.Reader == nil {
		opts.Reader = os.Stdin
	}
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	if opts.ErrWriter == nil {
		opts.ErrWriter = os.Stderr
	}
	if opts.UsageFunc == nil {
		opts.UsageFunc = defaultUsageFunc
	}
}

// Command ...
type Command struct {
	Usage       string
	Help        string
	Flags       []Flag
	Exec        func(*Context) error
	Subcommands []*Command
	Opts        *Options

	fs     *pflag.FlagSet
	parsed *Command
	parent *Command
}

// Build ...
func (c *Command) Build() error {
	if c.Exec == nil && len(c.Subcommands) == 0 {
		return &ErrMisconfigured{cmd: c, msg: "must define either exec or subcommands"}
	}
	if c.Exec != nil && len(c.Subcommands) > 0 {
		return &ErrMisconfigured{cmd: c, msg: "cannot define both exec and subommands"}
	}
	if c.Opts == nil {
		c.Opts = &Options{}
	}
	c.Opts.complete()
	for _, subcommand := range c.Subcommands {
		if subcommand.parent != nil {
			continue
		}
		if err := subcommand.setParent(c); err != nil {
			return err
		}
	}
	return nil
}

// Parse ...
func (c *Command) Parse(args []string) error {
	if err := c.Build(); err != nil {
		return err
	}

	// Check if the first argument was a subcommand
	if len(args) > 0 {
		for _, subcommand := range c.Subcommands {
			if subcommand.name() == args[0] {
				c.parsed = subcommand
				return subcommand.Parse(args[1:])
			}
		}
	}

	c.fs = pflag.NewFlagSet(c.name(), pflag.ContinueOnError)

	// Supress usage since pflag always prints help message if -h
	// or --help is parsed. We need to parse subcommands first.
	c.fs.Usage = func() {}

	if err := ParseFlags(c.fs, c.Flags, args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(c.Opts.ErrWriter, c.Opts.UsageFunc(c))
		}
		return err
	}

	c.parsed = c
	return nil
}

// Execute ...
func (c *Command) Execute(args []string) error {
	if err := c.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parsing command: %w", err)
	}
	return c.parsed.Exec(&Context{c.parsed.fs})
}

// name returns the name of the command.
func (c *Command) name() string {
	return strings.Split(c.Usage, " ")[0]
}

// usage returns the command.Usage prefixed by the command path of the parent command.
func (c *Command) usage() string {
	if p := c.parentPath(); p != "" {
		return p + " " + c.Usage
	}
	return c.Usage
}

// parentPath recurses up the command tree to construct the complete command path of the parent
func (c *Command) parentPath() string {
	if c.parent != nil {
		if path := c.parent.parentPath(); path != "" {
			return path + " " + c.parent.name()
		}
		return c.parent.name()
	}
	return ""
}

// setParent configures the parent for the current command.
func (c *Command) setParent(parent *Command) error {
	c.parent = parent
	if c.Opts == nil {
		c.Opts = c.parent.Opts
	}
	c.Flags = append(c.Flags, c.parent.Flags...)
	return nil
}

// defaultUsageFunc is the default function used to produce the usage string that is printed when
// -h or --help is specified by the user. It is the default value for UsageFunc in Options.
func defaultUsageFunc(c *Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: %s\n", c.usage())

	if len(c.Subcommands) > 0 {
		fmt.Fprint(&b, "\nAvailable Commands:\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 8, ' ', 0)
		for _, subcommand := range c.Subcommands {
			fmt.Fprintf(tw, "  %s\t%s\n", subcommand.name(), subcommand.Help)
		}
		tw.Flush()
	}

	if len(c.Flags) > 0 {
		fmt.Fprintf(&b, "\nFlags:\n%s\n", c.fs.FlagUsages())
	}

	return b.String()
}
