// +build ignore

package main

import (
	"os"
	"text/template"
)

func main() {
	f, err := os.Create("flag_types.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = flagTemplate.Execute(f, map[string]string{
		"Bool":          "bool",
		"BoolSlice":     "[]bool",
		"Duration":      "time.Duration",
		"DurationSlice": "[]time.Duration",
		"Int":           "int",
		"IntSlice":      "[]int",
		"String":        "string",
		"StringSlice":   "[]string",
	})
	if err != nil {
		panic(err)
	}
}

var flagTemplate = template.Must(template.New("").Parse(`package cli

// Code generated by go generate; DO NOT EDIT.

import (
	"time"

	"github.com/spf13/pflag"
)
{{ range $name, $type := . }}
var _ Flag = &{{ $name }}Flag{}

// {{ $name }}Flag is used to define a pflag.FlagSet.{{ $name }}P flag.
type {{ $name }}Flag struct {
	Name     string
	Usage    string
	EnvVar   []string
	Value    {{ $type }}
	Required bool
}

// Apply implements Flag.
func (f *{{ $name }}Flag) Apply(fs *pflag.FlagSet) {
	fs.{{ $name }}VarP(&f.Value, f.GetName(), f.GetShorthand(), f.Value, usageWithEnvVar(f.GetUsage(), f.GetEnvVar()))
}

// GetName implements Flag.
func (f *{{ $name }}Flag) GetName() string {
	s, _ := splitFlagName(f.Name)
	return s
}

// GetShorthand implements Flag.
func (f *{{ $name }}Flag) GetShorthand() string {
	_, s := splitFlagName(f.Name)
	return s
}

// GetUsage implements Flag.
func (f *{{ $name }}Flag) GetUsage() string {
	return f.Usage
}

// GetEnvVar implements Flag.
func (f *{{ $name }}Flag) GetEnvVar() []string {
	return f.EnvVar
}

// IsRequired implements Flag.
func (f *{{ $name }}Flag) IsRequired() bool {
	return f.Required
}
{{ end -}}
`))