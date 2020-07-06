## CLI

A minimal package for building CLIs in Go, built around [spf13/pflag](https://github.com/spf13/pflag) with inspiration from existing CLI packages.

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/itsdalmo/cli)
[![latest release](https://img.shields.io/github/v/release/itsdalmo/cli?style=flat-square)](https://github.com/itsdalmo/cli/releases/latest)
[![build status](https://img.shields.io/github/workflow/status/itsdalmo/cli/test?label=build&logo=github&style=flat-square)](https://github.com/itsdalmo/cli/actions?query=workflow%3Atest)
[![code quality](https://goreportcard.com/badge/github.com/itsdalmo/cli?style=flat-square)](https://goreportcard.com/report/github.com/itsdalmo/cli)

#### TODO
- [x] Add validation for redeclared flag names (and shorthand) in subcommands (instead of pflag panic).
- [x] Print global flags in a separate section under usage.
- [ ] Generate more flag types :D
- [ ] Validate arguments based on Usage? E.g. `command <in> <out>` could validate that two positional arguments exist during parse?
