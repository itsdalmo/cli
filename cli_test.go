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
		Opts: cli.Options{
			ErrWriter: os.Stdout,
		},
	}
	if err := c.Execute([]string{"--help"}); err != nil {
		panic(err)
	}
	// Output:
	//
	// Usage:
	//   printer [arg...]
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
		Opts: cli.Options{
			ErrWriter: os.Stdout,
		},
	}
	if err := c.Execute([]string{"--help"}); err != nil {
		panic(err)
	}
	// Output:
	//
	// Usage:
	//   printer [flags] [arg...]
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
					&cli.StringFlag{
						Name:  "delimiter",
						Usage: "Delimiter to use when printing",
						Value: "\n",
					},
				},
				Exec: func(c *cli.Context) error {
					times, err := c.GetInt("times")
					if err != nil {
						return err
					}
					delimiter, err := c.GetString("delimiter")
					for i := 0; i < times; i++ {
						fmt.Print(c.Arg(0), delimiter)
					}
					return nil
				},
			},
		},
		Opts: cli.Options{
			ErrWriter: os.Stdout,
		},
	}

	if err := c.Execute([]string{"repeat", "--help"}); err != nil {
		panic(err)
	}
	// Output:
	//
	// Repeatedly print the given argument
	//
	// Usage:
	//   printer repeat <arg>
	//
	// Flags:
	//       --delimiter string   Delimiter to use when printing (default "\n")
	//   -t, --times int          Number of times to print the argument [$PRINTER_REPEAT_TIMES] (default 3)
	//
	// Global Flags:
	//   -d, --debug   Enable debug logging
}

func Test_Subcommands_InheritGlobalFlags(t *testing.T) {
	c := cli.Command{
		Usage: "root [flags] [command]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug, d",
				Usage: "Enable debug logging",
			},
		},
		Subcommands: []*cli.Command{
			{
				Usage: "subcommand [flags]",
				Exec: func(c *cli.Context) error {
					debug, err := c.GetBool("debug")
					eq(t, nil, err)
					eq(t, true, debug)
					eq(t, 0, len(c.Args()))
					return nil
				},
			},
		},
	}

	if err := c.Execute([]string{"subcommand", "--debug"}); err != nil {
		t.Errorf("execute error: %s", err)
	}
}

func Test_Subcommands_IgnoresGlobalFlagOrder(t *testing.T) {
	c := cli.Command{
		Usage: "root [flags] [command]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug, d",
				Usage: "Enable debug logging",
			},
		},
		Subcommands: []*cli.Command{
			{
				Usage: "subcommand [flags]",
				Exec: func(c *cli.Context) error {
					debug, err := c.GetBool("debug")
					eq(t, nil, err)
					eq(t, true, debug)
					return nil
				},
			},
		},
	}

	if err := c.Execute([]string{"--debug", "subcommand"}); err != nil {
		t.Errorf("execute error: %s", err)
	}
}

func Test_NestedSubcommands(t *testing.T) {
	c := cli.Command{
		Usage: "root [flags] [command]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug, d",
				Usage: "Enable debug logging",
			},
		},
		Subcommands: []*cli.Command{
			{
				Usage: "nested",
				Subcommands: []*cli.Command{
					{
						Usage: "subcommand",
						Exec: func(c *cli.Context) error {
							debug, err := c.GetBool("debug")
							eq(t, nil, err)
							eq(t, true, debug)
							return nil
						},
					},
				},
			},
		},
	}

	if err := c.Execute([]string{"--debug", "nested", "subcommand"}); err != nil {
		t.Errorf("execute error: %s", err)
	}
}

func eq(t *testing.T, expected, got interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("\nexpected:\n%v\n\ngot:\n%v", expected, got)
	}
}
