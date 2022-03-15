package cli

import (
	"errors"
	"strings"

	"github.com/spf13/pflag"
)

// ErrHelp is returned if the --help flag was encountered during parsing.
var ErrHelp = errors.New("cli: help requested")

// ErrUnknownFlag is returned if an unknown flag is encountered during parsing.
type ErrUnknownFlag struct {
	cause error
	args  []string
}

// Error returns the cause error string.
func (e ErrUnknownFlag) Error() string {
	return e.cause.Error()
}

// Is returns true if the target error is an ErrUnknownFlag.
func (e ErrUnknownFlag) Is(target error) bool {
	_, ok := target.(ErrUnknownFlag)
	return ok
}

// Unparsed returns any unparsed args (including the unknown flag).
func (e ErrUnknownFlag) Unparsed() []string {
	// unknown shorthand flag: 'u' in -u (actually: -tu)
	// unknown shorthand flag: 'u' in -ut
	var failedArg string
	if i := strings.Index(e.cause.Error(), "-"); i > 0 {
		failedArg = e.cause.Error()[i:]
	}

	for i, arg := range e.args {
		if arg == failedArg {
			return e.args[i:]
		}
	}

	// For shorthand flags, the error message only contains the full argument
	// if the first shorthand is the unknown flag. I.e. more parsing is required.
	if isUnknownShorthand(e.cause) {
		name := strings.TrimPrefix(failedArg, "-")
		for i, arg := range e.args {
			if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
				continue
			}
			if ii := strings.Index(arg, name); ii > 0 {
				a := "-" + arg[ii:]
				return append([]string{a}, e.args[i+1:]...)
			}
		}
	}

	return []string{}
}

// Parse takes a list of flags and parses them from the provided arguments, using
// flag resolvers as a fallback. It returns the remaining arguments after all flags
// have been parsed.
func Parse(flags []Flag, resolvers []FlagResolver, args []string) ([]string, error) {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)

	for _, f := range flags {
		f.Apply(fs)
	}
	fs.Usage = func() {}

	var parseError error
	if err := fs.Parse(args); err != nil {
		switch {
		case isUnknownFlag(err) || isUnknownShorthand(err):
			parseError = ErrUnknownFlag{cause: err, args: args}
		case errors.Is(err, pflag.ErrHelp):
			parseError = ErrHelp
		default:
			parseError = err
		}
	}

	if err := ResolveMissingFlags(fs, flags, resolvers...); err != nil {
		return nil, err
	}

	return fs.Args(), parseError
}

// newFS returns a new pflag.FlagSet with the provided flags.
func newFS(flags []Flag) *pflag.FlagSet {
	fs := pflag.NewFlagSet("", pflag.ContinueOnError)
	for _, f := range flags {
		f.Apply(fs)
	}
	fs.Usage = func() {}
	return fs
}

func isUnknownFlagErr(e error) bool {
	return isUnknownFlag(e) || isUnknownShorthand(e)
}

// isUnknown flag returns true if the pflag.Parse error is due to an unknown flag.
func isUnknownFlag(e error) bool {
	return strings.HasPrefix(e.Error(), "unknown flag")
}

// isUnknownShorthand returns true if the pflag.Parse error is due to an unknown shorthand flag.
func isUnknownShorthand(e error) bool {
	return strings.HasPrefix(e.Error(), "unknown shorthand flag")
}
