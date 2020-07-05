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

// FlagResolver is the interface implemented by custom flag resolvers.
type FlagResolver interface {
	Resolve(Flag) (string, bool)
}

// EnvVarResolver implements FlagResolver by resolving variables from the environment.
type EnvVarResolver struct{}

// Resolve implements FlagResolver.
func (*EnvVarResolver) Resolve(flag Flag) (string, bool) {
	for _, k := range flag.GetEnvVar() {
		v, found := os.LookupEnv(strings.TrimPrefix(k, "$"))
		if found {
			return v, found
		}
	}
	return "", false
}

// ResolveMissingFlags iterates over all missing flags in the given pflag.FlagSet and applies each FlagResolver in turn
// until the the flag is resolved. An error is returned if we are unable to set the flag to the resolved value, or if
// a required Flag has missing values after applying all resolvers.
func ResolveMissingFlags(fs *pflag.FlagSet, flags []Flag, resolvers ...FlagResolver) error {
	var (
		missingFlags []string
		resolverErr  error
	)

	fs.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			return // Flag has been set via commandline
		}
		for _, flag := range flags {
			if flag.GetName() != f.Name {
				continue
			}
			var (
				found bool
				value string
			)
			for _, resolver := range resolvers {
				value, found = resolver.Resolve(flag)
				if found {
					err := f.Value.Set(value)
					if err != nil {
						resolverErr = err
					}
					break // Flag was resolved
				}
			}
			if !found && flag.IsRequired() {
				missingFlags = append(missingFlags, flag.GetName())
			}
		}
	})
	if resolverErr != nil {
		return resolverErr
	}
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
