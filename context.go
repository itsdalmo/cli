package cli

import "fmt"

// Context is passed to the user defined exec function when a command has been parsed.
type Context struct {
	args  []string
	flags map[string]Flag
}

// Args returns the remaining arguments after the command has been parsed.
func (c *Context) Args() []string {
	return c.args
}

// Arg returns the i'th argument.
func (c *Context) Arg(i int) string {
	return c.args[i]
}

// NArg returns the number of arguments passed to the command.
func (c *Context) NArg() int {
	return len(c.args)
}

// lookup a flag by its long name. Panics if the flag by that name has not been
// defined on the context.
func (c *Context) lookup(name string) Flag {
	f, ok := c.flags[name]
	if !ok {
		panic(fmt.Errorf("flag not defined: %q", name))
	}
	return f
}

// typeMismatchErr is thrown (as a panic) if the wrong getter to reterieve a flag
// value. This is a program error and therefor should panic.
func typeMismatchErr(name, want string, value interface{}) error {
	return fmt.Errorf("type mismatch for flag: %q (%s != %T)", name, want, value)
}
