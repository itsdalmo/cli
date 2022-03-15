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
	Opts        Options

	fs     *pflag.FlagSet
	parent *Command
}

// initialize ...
func (c *Command) initialize() (err error) {
	if c.Usage == "" {
		return &ErrMisconfigured{cmd: c, msg: "usage must be defined"}
	}
	if c.Exec == nil && len(c.Subcommands) == 0 {
		return &ErrMisconfigured{cmd: c, msg: "must define either exec or subcommands"}
	}
	if c.Exec != nil && len(c.Subcommands) > 0 {
		return &ErrMisconfigured{cmd: c, msg: "cannot define both exec and subcommands"}
	}
	// TODO: Ensure that options can only be set on the root command.
	c.Opts.complete()

	c.fs = newFS(c.LocalFlags())
	if c.parent != nil {
		// TODO: Ensure that subcommands cannot redefine a global flag. Then
		// get rid of this line.
		c.fs.AddFlagSet(c.parent.fs)
	}

	for _, subcommand := range c.Subcommands {
		if err := subcommand.setParent(c); err != nil {
			return err
		}
	}
	return nil
}

func (c *Command) LocalFlags() []Flag {
	return c.Flags
}

func (c *Command) GlobalFlags() []Flag {
	var fs []Flag
	if c.parent != nil {
		fs = append(fs, c.parent.CombinedFlags()...)
	}
	return fs
}

func (c *Command) CombinedFlags() []Flag {
	fs := c.LocalFlags()
	if c.parent != nil {
		fs = append(fs, c.parent.CombinedFlags()...)
	}
	return fs
}

// parse ...
func (c *Command) parse(args []string) (*Command, error) {
	if err := c.initialize(); err != nil {
		return nil, err
	}
	var (
		parseError    error
		unparsed      []string
		helpRequested bool
	)
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
						unparsed = args[ii:]
						break
					}
				}
			}
			parseError = err
		case errors.Is(err, pflag.ErrHelp):
			// Wait with returning error until we have checked arguments to see if --help was specified for a subcommand.
			parseError, helpRequested = err, true
		default:
			return nil, err
		}
	}

	if err := ResolveMissingFlags(c.fs, c.Flags, c.Opts.Resolvers...); err != nil {
		return nil, err
	}

	if len(c.Subcommands) > 0 {
		for _, subcommand := range c.Subcommands {
			if subcommand.name() == c.fs.Arg(0) {
				args = append(c.fs.Args()[1:], unparsed...)

				cmd, err := subcommand.parse(args)
				if err != nil {
					return cmd, err
				}
				if helpRequested {
					return cmd, parseError
				}
				return cmd, nil
			}
		}
		if !helpRequested {
			parseError = errors.New("no subcommand specified. See --help")
		}
	}

	return c, parseError
}

// Execute ...
func (c *Command) Execute(args []string) error {
	cmd, err := c.parse(args)
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintln(cmd.Opts.ErrWriter, cmd.Opts.UsageFunc(cmd))
			return nil
		}
		return fmt.Errorf("parsing command: %w", err)
	}

	ctx := &Context{args: cmd.fs.Args(), flags: make(map[string]Flag)}

	for _, f := range cmd.CombinedFlags() {
		ctx.flags[f.GetName()] = f
	}

	return cmd.Exec(ctx)
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
func (c *Command) setParent(parent *Command) error {
	c.parent, c.Opts = parent, parent.Opts
	return nil
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

	if flags := c.LocalFlags(); len(flags) > 0 {
		fmt.Fprintf(&b, "\nFlags:\n%s", newFS(flags).FlagUsages())
	}

	if flags := c.GlobalFlags(); len(flags) > 0 {
		fmt.Fprintf(&b, "\nGlobal Flags:\n%s", newFS(flags).FlagUsages())
	}

	return b.String()
}
