package cli_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/itsdalmo/cli"
)

func TestParse(t *testing.T) {
	flags := []cli.Flag{&cli.StringFlag{Name: "test, t"}}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "normal flag",
			args: []string{"--test", "flag", "arg"},
			want: []string{"arg"},
		},
		{
			name: "shorthand flag",
			args: []string{"-t", "flag", "arg"},
			want: []string{"arg"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := cli.Parse(flags, nil, tc.args)
			eq(t, nil, err)
			eq(t, tc.want, got)
		})
	}
}

func TestParseUnknownFlag(t *testing.T) {
	flags := []cli.Flag{&cli.BoolFlag{Name: "test, t"}}

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flag",
			args: []string{"--test", "--unknown", "flag"},
			want: []string{"--unknown", "flag"},
		},
		{
			name: "shorthand",
			args: []string{"-t", "-u", "flag"},
			want: []string{"-u", "flag"},
		},
		{
			name: "chained shorthands",
			args: []string{"-tu", "flag1"},
			want: []string{"-tu", "flag1"},
		},
		{
			name: "chained shorthands (wrong order)",
			args: []string{"-ut", "flag1"},
			want: []string{"-ut", "flag1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cli.Parse(flags, nil, tc.args)

			var euf cli.ErrUnknownFlag
			if !errors.As(err, &euf) {
				t.Error("expected unknown flag error")
			}
			fmt.Println(euf.Error())
			eq(t, tc.want, euf.Unparsed())
		})
	}
}
