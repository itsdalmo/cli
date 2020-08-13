package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/itsdalmo/cli"
)

func main() {
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
				Usage: "echo [flags] [<arg>...]",
				Help:  "Echo the specified args",
				Exec:  echo,
			},
			{
				Usage: "repeat <arg>",
				Help:  "Repeatedly print the given argument",
				Exec:  repeat,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:   "times, t",
						Usage:  "Number of times to print the argument",
						Value:  3,
						EnvVar: []string{"PRINTER_REPEAT_TIMES"},
					},
				},
			},
		},
	}
	if err := c.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(c.Opts.ErrWriter, err)
		os.Exit(1)
	}
}

func echo(c *cli.Context) error {
	fmt.Println(strings.Join(c.Args(), " "))
	return nil
}

func repeat(c *cli.Context) error {
	times, err := c.GetInt("times")
	if err != nil {
		return err
	}
	for i := 0; i < times; i++ {
		fmt.Println(c.Arg(0))
	}
	return nil
}
