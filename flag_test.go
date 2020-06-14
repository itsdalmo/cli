package cli_test

import (
	"errors"
	"os"
	"testing"

	"github.com/itsdalmo/cli"
)

func TestFlag(t *testing.T) {
	var f cli.Flag = &cli.StringFlag{
		Name:     "region, r",
		Usage:    "AWS Region to target",
		EnvVar:   []string{"AWS_REGION", "AWS_DEFAULT_REGION"},
		Required: true,
	}

	eq(t, "region", f.GetName())
	eq(t, "r", f.GetShorthand())
	eq(t, `AWS Region to target`, f.GetUsage())
	eq(t, []string{"AWS_REGION", "AWS_DEFAULT_REGION"}, f.GetEnvVar())
	eq(t, true, f.IsRequired())
}

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		description       string
		args              []string
		env               map[string]string
		expectedRegion    string
		expectedInstances []string
		expectedErr       error
	}{
		{
			description:       "works",
			args:              []string{"--region", "eu-north-1", "-i", "i-1", "-i", "i-2"},
			expectedRegion:    "eu-north-1",
			expectedInstances: []string{"i-1", "i-2"},
		},
		{
			description:       "supports environment variables",
			env:               map[string]string{"AWS_REGION": "eu-north-1", "AWS_INSTANCES": "i-1,i-2,i-3"},
			expectedRegion:    "eu-north-1",
			expectedInstances: []string{"i-1", "i-2", "i-3"},
		},
		{
			description: "errors if expected flag is missing",
			expectedErr: errors.New("missing required flags [region]"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			c := cli.Command{
				Usage: "echo [flags] [<arg>...]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "region, r",
						Usage:    "AWS Region to target",
						EnvVar:   []string{"AWS_REGION"},
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:   "instance, i",
						Usage:  "An instance to target",
						EnvVar: []string{"AWS_INSTANCES"},
					},
				},
				Exec: func(c *cli.Context) error {
					region, err := c.GetString("region")
					if err != nil {
						t.Fatalf("unexpected error: %s", err)
					}
					eq(t, tc.expectedRegion, region)

					instances, err := c.GetStringSlice("instance")
					if err != nil {
						t.Fatalf("unexpected error: %s", err)
					}
					for i, expected := range tc.expectedInstances {
						eq(t, expected, instances[i])
					}
					return nil
				},
			}

			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatal(err)
				}
				defer os.Unsetenv(k)
			}

			err := c.Execute(tc.args)
			eq(t, tc.expectedErr, errors.Unwrap(err))

		})
	}
}
