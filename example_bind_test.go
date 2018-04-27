package v8_test

import (
	"fmt"
	"strconv"

	"github.com/augustoroman/v8"
)

// AddAllNumbers is the callback function that we'll make accessible the JS VM.
// It will accept 2 or more numbers and return the sum. If fewer than two args
// are passed or any of the args are not parsable as numbers, it will fail.
func AddAllNumbers(in v8.CallbackArgs) (*v8.Value, error) {
	if len(in.Args) < 2 {
		return nil, fmt.Errorf("add requires at least 2 numbers, but got %d args", len(in.Args))
	}
	result := 0.0
	for i, arg := range in.Args {
		n, err := strconv.ParseFloat(arg.String(), 64)
		if err != nil {
			return nil, fmt.Errorf("Arg %d [%q] cannot be parsed as a number: %v", i, arg.String(), err)
		}
		result += n
	}
	return in.Context.Create(result)
}

func ExampleContext_Bind() {
	ctx := v8.NewIsolate().NewContext()

	// First, we'll bind our callback function into a *v8.Value that we can
	// use as we please. The string "my_add_function" here is the used by V8 as
	// the name of the function. That is, we've defined:
	//   val.toString() = (function my_add_function() { [native code] });
	// However the name "my_add_function" isn't actually accessible in the V8
	// global scope anywhere yet.
	val := ctx.Bind("my_add_function", AddAllNumbers)

	// Next we'll set that value into the global context to make it available to
	// the JS.
	if err := ctx.Global().Set("add", val); err != nil {
		panic(err)
	}

	// Now we'll call it!
	result, err := ctx.Eval(`add(1,2,3,4,5)`, `example.js`)
	if err != nil {
		panic(err)
	}
	fmt.Println(`add(1,2,3,4,5) =`, result)

	// output:
	// add(1,2,3,4,5) = 15
}
