package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/pflag"
)

// Flag is the interface implemented by all flag types.
//
//go:generate go run flag_gen.go
type Flag interface {
	// Apply the flag to the given flagset.
	Apply(fs *pflag.FlagSet)

	// GetName returns the name of the flag.
	GetName() string

	// GetShorthand returns the shorthand of the flag (if it is defined).
	GetShorthand() string

	// GetUsage returns the usage for the flag.
	GetUsage() string

	// GetEnvVar returns the env variables used to set this flag.
	GetEnvVar() []string

	// IsRequired returns true if the flag is marked as required.
	IsRequired() bool
}

// ParseFlags takes a list of flags and applies them to the provided pflag.Flagset
// before parsing the flagset. Environment variables will be used to set flag values
// if they are defined, and the flag was not passed on the commandline.
func ParseFlags(fs *pflag.FlagSet, flags []Flag, args []string) error {
	for _, f := range flags {
		f.Apply(fs)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	var missingFlags []string

	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return // Flag has been set via commandline
		}
		for _, flag := range flags {
			if flag.GetName() == f.Name {
				var (
					value string
					found bool
				)
				for _, k := range flag.GetEnvVar() {
					value, found = os.LookupEnv(strings.TrimPrefix(k, "$"))
					if found {
						f.Value.Set(value)
						break // Flag was set via environment
					}
				}
				if !found && flag.IsRequired() {
					missingFlags = append(missingFlags, flag.GetName())
				}
			}
		}
	})
	if len(missingFlags) > 0 {
		return fmt.Errorf("missing required flags %v", missingFlags)
	}
	return nil
}

func usageWithEnvVar(usage string, vars []string) string {
	if len(vars) == 0 {
		return usage
	}
	varPrefix, varSuffix := "$", ""
	if runtime.GOOS == "windows" {
		varPrefix, varSuffix = "%", "%"
	}
	for i, v := range vars {
		vars[i] = varPrefix + v + varSuffix
	}
	return fmt.Sprintf("%s [%s]", usage, strings.Join(vars, ", "))
}

func splitFlagName(name string) (longName string, shortName string) {
	splits := strings.Split(name, ",")
	switch len(splits) {
	default:
		panic(fmt.Errorf("invalid variable name: %s", name))
	case 2:
		shortName = splits[1]
		fallthrough
	case 1:
		longName = splits[0]
	}
	return strings.TrimSpace(longName), strings.TrimSpace(shortName)
}
