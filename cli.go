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
	Resolvers []FlagResolver
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
	if opts.Resolvers == nil {
		opts.Resolvers = []FlagResolver{&EnvVarResolver{}}
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
	gfs    *pflag.FlagSet
	parent *Command
	parsed *Command
}

// Setup ...
func (c *Command) Setup() (err error) {
	if c.Usage == "" {
		return &ErrMisconfigured{cmd: c, msg: "usage must be defined"}
	}
	if c.Exec == nil && len(c.Subcommands) == 0 {
		return &ErrMisconfigured{cmd: c, msg: "must define either exec or subcommands"}
	}
	if c.Exec != nil && len(c.Subcommands) > 0 {
		return &ErrMisconfigured{cmd: c, msg: "cannot define both exec and subcommands"}
	}
	if c.Opts == nil {
		c.Opts = &Options{}
	}
	c.Opts.complete()

	// Reset flagset each time setup is called.
	c.fs = pflag.NewFlagSet(c.name(), pflag.ContinueOnError)
	c.fs.Usage = func() {}

	// Apply flags to flagset.
	for _, flag := range c.Flags {
		flag.Apply(c.fs)
	}

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
func (c *Command) Parse(args []string) (parseError error) {
	if err := c.Setup(); err != nil {
		return err
	}
	var helpRequested bool
	if err := c.fs.Parse(args); err != nil {
		switch {
		case isUnknownFlagErr(err):
			// Unknown flags might belong to a subcommand so we wait to return. We should remove arguments that have
			// been successfully parsed, which can be done somewhat hackily by parsing the name of the flag from the
			// error message.
			if i := strings.Index(err.Error(), "-"); i > 0 {
				failedArg := err.Error()[i:]
				for ii, arg := range args {
					if arg == failedArg {
						args = args[(ii - 1):]
						break
					}
				}
			}
			parseError = err
		case errors.Is(err, pflag.ErrHelp):
			// Wait with returning error until we have checked arguments to see if --help was specified for a subcommand.
			parseError, helpRequested = err, true
		default:
			return err
		}
	}

	if err := ResolveMissingFlags(c.fs, c.Flags, c.Opts.Resolvers...); err != nil {
		return err
	}

	if len(c.Subcommands) > 0 {
		for _, subcommand := range c.Subcommands {
			if subcommand.name() == c.fs.Arg(0) {
				c.parsed = subcommand
				return subcommand.Parse(args[1:])
			}
		}
		if !helpRequested {
			parseError = errors.New("no subcommand specified. See --help")
		}
	}

	c.parsed = c
	return parseError
}

// Execute ...
func (c *Command) Execute(args []string) error {
	if err := c.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(c.parsed.Opts.ErrWriter, c.parsed.Opts.UsageFunc(c.parsed))
			return nil
		}
		return fmt.Errorf("parsing command: %w", err)
	}
	flags, err := c.parsed.combinedFlags()
	if err != nil {
		return err
	}
	return c.parsed.Exec(&Context{flags})
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

// parentPath recurses up the command tree to construct the complete command path of the parent.
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
func (c *Command) setParent(parent *Command) (err error) {
	c.parent = parent
	if c.Opts == nil {
		c.Opts = c.parent.Opts
	}
	c.gfs, err = parent.combinedFlags()
	if err != nil {
		return err
	}
	return nil
}

// combinedFlags returns a pflag.FlagSet containing both local and global flags for the command.
func (c *Command) combinedFlags() (*pflag.FlagSet, error) {
	if c.gfs == nil {
		return c.fs, nil
	}

	var redefined []string
	c.gfs.VisitAll(func(f *pflag.Flag) {
		if c.fs.Lookup(f.Name) != nil || c.fs.ShorthandLookup(f.Shorthand) != nil {
			redefined = append(redefined, f.Name)
			return
		}
		c.fs.AddFlag(f)
	})
	if len(redefined) > 0 {
		return nil, &ErrMisconfigured{cmd: c, msg: fmt.Sprintf("global flags redefined locally: %v", redefined)}
	}
	return c.fs, nil
}

// isUnknownFlagErr returns true if the given pflag.Parse error is due to an unknown flag or shorthand.
func isUnknownFlagErr(e error) bool {
	return strings.HasPrefix(e.Error(), "unknown flag") || strings.HasPrefix(e.Error(), "unknown shorthand flag")
}

// defaultUsageFunc is the default function used to produce the usage string that is printed when
// -h or --help is specified by the user. It is the default value for UsageFunc in Options.
func defaultUsageFunc(c *Command) string {
	var b strings.Builder

	if c.Help != "" {
		fmt.Fprint(&b, c.Help, "\n\n")
	}

	fmt.Fprintf(&b, "Usage:\n  %s\n", c.usage())

	if len(c.Subcommands) > 0 {
		fmt.Fprint(&b, "\nAvailable Commands:\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 8, ' ', 0)
		for _, subcommand := range c.Subcommands {
			fmt.Fprintf(tw, "  %s\t%s\n", subcommand.name(), subcommand.Help)
		}
		tw.Flush()
	}

	if c.fs.HasAvailableFlags() {
		fmt.Fprintf(&b, "\nFlags:\n%s", c.fs.FlagUsages())
	}

	if c.gfs != nil && c.gfs.HasAvailableFlags() {
		fmt.Fprintf(&b, "\nGlobal Flags:\n%s", c.gfs.FlagUsages())
	}

	return b.String()
}
