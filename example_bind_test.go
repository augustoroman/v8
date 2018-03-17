package v8

import (
	"fmt"
)

type Console struct {
	Info Callback // Both exported variables and exported methods work!
}

func (c Console) Log(in CallbackArgs) (*Value, error) {
	fmt.Printf("%s:%d>", in.Caller.Filename, in.Caller.Line)
	for _, arg := range in.Args {
		fmt.Print(" ", arg)
	}
	fmt.Print("\n")
	return nil, nil
}

func ExampleContext_Bind() {
	ctx := NewIsolate().NewContext()

	var c Console
	c.Info = c.Log // bind Info to c.Log for a compact example.

	consoleOb, err := ctx.Create(c)
	failOnError(err)
	failOnError(ctx.Global().Set("console", consoleOb))

	_, err = ctx.Eval(`
        console.Log('Hello', 'World!');
        console.Info(function() { return "x" });
        console.Log([1,2,3]);
    `, "hello.js")
	failOnError(err)

	// output:
	// hello.js:2> Hello World!
	// hello.js:3> function() { return "x" }
	// hello.js:4> 1,2,3
}

func failOnError(err error) {
	if err != nil {
		panic(err)
	}
}
