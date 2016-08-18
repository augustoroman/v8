// v8-runjs is a command-line tool to run javascript files.
//
// It's like node, but less useful.
//
// It runs the javascript files provided on the commandline in order until
// it finishes or an error occurs.
//
// Other than the standard javascript environment, it provides console.*:
//   console.log, console.info: write args to stdout
//   console.warn:              write args to stderr in yellow
//   console.error:             write args to stderr in scary red
//
// Sooo... you can run your JS and print to the screen.
package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/augustoroman/v8"
)

var console Console

func main() {
	flag.Parse()
	ctx := v8.NewIsolate().NewContext()
	consoleOb, err := ctx.Create(map[string]interface{}{
		"log":   console.Info,
		"info":  console.Info,
		"warn":  console.Warn,
		"error": console.Error,
	})
	failOnError(err)
	failOnError(ctx.Global().Set("console", consoleOb))

	for _, filename := range flag.Args() {
		data, err := ioutil.ReadFile(filename)
		failOnError(err)
		_, err = ctx.Eval(string(data), filename)
		failOnError(err)
	}
}

func failOnError(err error) {
	if err != nil {
		console.writeLog(os.Stderr, RED, err)
		os.Exit(1)
	}
}
