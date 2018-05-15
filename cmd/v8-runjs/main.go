// v8-runjs is a command-line tool to run javascript.
//
// It's like node, but less useful.
//
// It runs the javascript files provided on the commandline in order until
// it finishes or an error occurs. If no files are provided, this will enter a
// REPL mode where you can interactively run javascript.
//
// Other than the standard javascript environment, it provides console.*:
//   console.log, console.info: write args to stdout
//   console.warn:              write args to stderr in yellow
//   console.error:             write args to stderr in scary red
//
// Sooo... you can run your JS and print to the screen.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/augustoroman/v8"
	"github.com/augustoroman/v8/v8console"
	"github.com/peterh/liner"
)

const (
	kRESET = "\033[0m"
	kRED   = "\033[91m"
)

func main() {
	flag.Parse()
	ctx := v8.NewIsolate().NewContext()
	v8console.Config{"", os.Stdout, os.Stderr, true}.Inject(ctx)

	var outstanding_tasks sync.WaitGroup

	ctx.Global().Set("sleep", ctx.Bind("sleep", func(args v8.CallbackArgs) (*v8.Value, error) {
		if len(args.Args) == 0 {
			return nil, errors.New("sleep requires duration parameter (in msec)")
		}
		dt := time.Duration(args.Arg(0).Float64() * float64(time.Millisecond))
		promise, _ := NewPromise(ctx)
		outstanding_tasks.Add(1)
		time.AfterFunc(dt, func() {
			promise.Resolve.Call(nil, args.Arg(0))
			outstanding_tasks.Done()
		})
		return promise.Value, nil
	}))

	for _, filename := range flag.Args() {
		data, err := ioutil.ReadFile(filename)
		failOnError(err)
		_, err = ctx.Eval(string(data), filename)
		failOnError(err)
	}

	if flag.NArg() == 0 {
		s := liner.NewLiner()
		s.SetMultiLineMode(true)
		defer s.Close()
		for {
			jscode, err := s.Prompt("> ")
			if err == io.EOF {
				break
			}
			failOnError(err)
			s.AppendHistory(jscode)
			result, err := ctx.Eval(jscode, "<input>")
			if err != nil {
				fmt.Println(kRED, err, kRESET)
			} else {
				fmt.Println(result)
			}
		}
	}

	// Wait for any outstanding promises to complete before exiting. Note that
	// this isn't quite correct: if a promise completes but, while computing
	// its resolution it creates _another_ promise, this will fail. We really
	// want a loop that waits until the semaphore has hit zero, but WaitGroup
	// doesn't expose the sempahore value.
	outstanding_tasks.Wait()
	ctx.Eval("", "done.js")
}

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}

// Helper to create and return a promise. This is non-optimal for several
// reasons:
//   - It's slow because it makes a bunch of cgo calls to construct the value.
//   - Every new promise binds a new function callback that is never released,
//     meaning that this is basically a memory leak.
//
// A better solution would be to allow directly accessing the
// v8::Promise::Resolver interface directly, so somehow you could create a
// promise and directly resolve() or reject() values.
//
// Also, fix the callback leak: https://github.com/augustoroman/v8/issues/29

type Promise struct{ Value, Resolve, Reject *v8.Value }

func NewPromise(ctx *v8.Context) (*Promise, error) {
	promise_class, err := ctx.Global().Get("Promise")
	if err != nil {
		return nil, fmt.Errorf("Cannot get Promise class: %v", err)
	}
	var p Promise
	p.Value, err = promise_class.New(ctx.Bind(
		"promise_handler",
		func(args v8.CallbackArgs) (*v8.Value, error) {
			p.Resolve, p.Reject = args.Arg(0), args.Arg(1)
			return nil, nil
		}))
	return &p, err
}
