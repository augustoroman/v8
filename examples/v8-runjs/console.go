package main

import (
	"fmt"
	"github.com/augustoroman/v8"
	"io"
	"os"
)

const RESET = "\033[0m"
const NO_COLOR = ""
const RED = "\033[91m"
const YELLOW = "\033[93m"

type Console struct{}

func (c Console) writeLog(w io.Writer, color string, vals ...interface{}) {
	if color != "" {
		fmt.Fprint(w, color)
	}
	fmt.Fprint(w, vals...)
	if color != "" {
		fmt.Fprint(w, RESET)
	}
	fmt.Fprint(w, "\n")
}
func (c Console) toInterface(vals []*v8.Value) []interface{} {
	out := make([]interface{}, len(vals))
	for i, val := range vals {
		out[i] = val
	}
	return out
}
func (c Console) toInterfaceWithLoc(caller v8.Loc, args []*v8.Value) []interface{} {
	var vals []interface{}
	vals = append(vals, fmt.Sprintf("[%s:%d] ", caller.Filename, caller.Line))
	vals = append(vals, c.toInterface(args)...)
	return vals
}
func (c Console) Info(caller v8.Loc, args ...*v8.Value) (*v8.Value, error) {
	c.writeLog(os.Stdout, NO_COLOR, c.toInterface(args)...)
	return nil, nil
}
func (c Console) Warn(caller v8.Loc, args ...*v8.Value) (*v8.Value, error) {
	c.writeLog(os.Stderr, YELLOW, c.toInterfaceWithLoc(caller, args)...)
	return nil, nil
}
func (c Console) Error(caller v8.Loc, args ...*v8.Value) (*v8.Value, error) {
	c.writeLog(os.Stderr, RED, c.toInterfaceWithLoc(caller, args)...)
	return nil, nil
}
