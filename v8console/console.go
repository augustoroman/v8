// Package v8console provides a simple console implementation to allow JS to
// log messages.
//
// It supports the console.log, console.info, console.warn, and console.error
// functions and logs the result of .ToString() on each of the arguments.  It
// can color warning and error messages, but does not support Chrome's fancy
// %c message styling.
package v8console

import (
	"fmt"
	"io"

	"github.com/augustoroman/v8"
)

const (
	kRESET    = "\033[0m"
	kNO_COLOR = ""
	kRED      = "\033[91m"
	kYELLOW   = "\033[93m"
)

// Config holds configuration for a particular console instance.
type Config struct {
	// Prefix to prepend to every log message.
	Prefix string
	// Destination for all .log and .info calls.
	Stdout io.Writer
	// Destination for all .warn and .error calls.
	Stderr io.Writer
	// Whether to enable ANSI color escape codes in the output.
	Colorize bool
}

// Inject sets the global "console" object of the specified Context to bind
// .log, .info, .warn, and .error to call this Console object.  If the console
// object already exists in the global namespace, only the log/info/warn/error
// properties are replaced.
func (c Config) Inject(ctx *v8.Context) {
	ob, _ := ctx.Global().Get("console")
	if ob == nil || ob.String() != "[object Object]" {
		// If the object doesn't already exist, create a new object from scratch
		// and inject the whole thing.
		ctx, err := ctx.Create(map[string]interface{}{
			"log":   c.Info,
			"info":  c.Info,
			"warn":  c.Warn,
			"error": c.Error,
		})
		if err != nil {
			// This should never happen: our map is well-defined for ctx.Create.
			panic(fmt.Errorf("cannot create ctx object: %v", err))
		}
		ob = ctx
	} else {
		// If the console object already exists, just replace the logging
		// methods.
		functions := []struct {
			name     string
			callback v8.Callback
		}{
			{"log", c.Info},
			{"info", c.Info},
			{"warn", c.Warn},
			{"error", c.Error},
		}
		for _, fn := range functions {
			if err := ob.Set(fn.name, ctx.Bind(fn.name, fn.callback)); err != nil {
				panic(fmt.Errorf("cannot set %s on console object: %v", fn.name, err))
			}
		}
	}

	// Update console object.
	if err := ctx.Global().Set("console", ob); err != nil {
		// This should never happen: Global() is always an object.
		panic(fmt.Errorf("cannot set context into global: %v", err))
	}
}

func (c Config) writeLog(w io.Writer, color string, vals ...interface{}) {
	if color != "" && c.Colorize {
		fmt.Fprint(w, color)
	}
	fmt.Fprint(w, c.Prefix)
	fmt.Fprint(w, vals...)
	if color != "" && c.Colorize {
		fmt.Fprint(w, kRESET)
	}
	fmt.Fprint(w, "\n")
}
func (c Config) toInterface(vals []*v8.Value) []interface{} {
	out := make([]interface{}, len(vals))
	for i, val := range vals {
		out[i] = val
	}
	return out
}
func (c Config) toInterfaceWithLoc(caller v8.Loc, args []*v8.Value) []interface{} {
	var vals []interface{}
	vals = append(vals, fmt.Sprintf("[%s:%d] ", caller.Filename, caller.Line))
	vals = append(vals, c.toInterface(args)...)
	return vals
}

// Info is the v8 callback function that is registered for the console.log and
// console.info functions.
func (c Config) Info(in v8.CallbackArgs) (*v8.Value, error) {
	c.writeLog(c.Stdout, kNO_COLOR, c.toInterface(in.Args)...)
	return nil, nil
}

// Warn is the v8 callback function that is registered for the console.warn
// functions.
func (c Config) Warn(in v8.CallbackArgs) (*v8.Value, error) {
	c.writeLog(c.Stderr, kYELLOW, c.toInterfaceWithLoc(in.Caller, in.Args)...)
	return nil, nil
}

// Error is the v8 callback function that is registered for the console.error
// functions.
func (c Config) Error(in v8.CallbackArgs) (*v8.Value, error) {
	c.writeLog(c.Stderr, kRED, c.toInterfaceWithLoc(in.Caller, in.Args)...)
	return nil, nil
}
