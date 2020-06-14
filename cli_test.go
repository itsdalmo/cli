package cli_test

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/itsdalmo/cli"
)

func Example_minimal() {
	c := cli.Command{
		Usage: "printer [arg...]",
		Exec: func(c *cli.Context) error {
			return nil
		},
		Opts: &cli.Options{
			ErrWriter: os.Stdout,
		},
	}
	if err := c.Execute([]string{"--help"}); err != nil {
		panic(err)
	}
	// Output:
	// Usage: printer [arg...]
}

func Example_basic() {
	c := cli.Command{
		Usage: "printer [flags] [arg...]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug, d",
				Usage: "Enable debug logging",
			},
		},
		Exec: func(c *cli.Context) error {
			debug, err := c.GetBool("debug")
			if err != nil {
				return err
			}
			if debug {
				fmt.Println("debugging is enabled")
			}
			return nil
		},
		Opts: &cli.Options{
			ErrWriter: os.Stdout,
		},
	}
	if err := c.Execute([]string{"--help"}); err != nil {
		panic(err)
	}
	// Output:
	// Usage: printer [flags] [arg...]
	//
	// Flags:
	//   -d, --debug   Enable debug logging
}

func Example_subcommands() {
	c := cli.Command{
		Usage: "printer [flags] [command]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug, d",
				Usage: "Enable debug logging",
			},
		},
		Subcommands: []*cli.Command{
			{
				Usage: "echo [flags] [arg...]",
				Help:  "Echo the specified args",
				Exec: func(c *cli.Context) error {
					fmt.Println(c.Args())
					return nil
				},
			},
			{
				Usage: "repeat <arg>",
				Help:  "Repeatedly print the given argument",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:   "times, t",
						Usage:  "Number of times to print the argument",
						Value:  3,
						EnvVar: []string{"PRINTER_REPEAT_TIMES"},
					},
				},
				Exec: func(c *cli.Context) error {
					times, err := c.GetInt("times")
					if err != nil {
						return err
					}
					for i := 0; i < times; i++ {
						fmt.Println(c.Arg(0))
					}
					return nil
				},
			},
		},
		Opts: &cli.Options{
			ErrWriter: os.Stdout,
		},
	}

	if err := c.Execute([]string{"repeat", "--help"}); err != nil {
		panic(err)
	}
	// Output:
	// Usage: printer repeat <arg>
	//
	// Flags:
	//   -d, --debug       Enable debug logging
	//   -t, --times int   Number of times to print the argument [$PRINTER_REPEAT_TIMES] (default 3)
}

func eq(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nexpected:\n%v\n\ngot:\n%v", expected, got)
	}
}
