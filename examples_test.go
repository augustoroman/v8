package v8_test

import (
	"fmt"

	"github.com/augustoroman/v8"
)

func Example() {
	// Easy-peasy to create a new VM:
	ctx := v8.NewIsolate().NewContext()

	// You can load your js from a file, create it dynamically, whatever.
	ctx.Eval(`
            // This is javascript code!
            add = (a,b)=>{ return a + b }; // whoa, ES6 arrow functions.
        `, "add.js") // <-- supply filenames for stack traces

	// State accumulates in a context.  Add still exists.
	// The last statements' value is returned to Go.
	res, _ := ctx.Eval(`add(3,4)`, "compute.js") // don't ignore errors!
	fmt.Println("add(3,4) =", res.String())      // I hope it's 7.

	// You can also bind Go functions to javascript:
	my_count_function := func(in v8.CallbackArgs) (*v8.Value, error) {
		return in.Context.Create(len(in.Args)) // ctx.Create is great for mapping Go -> JS.
	}
	cnt := ctx.Bind("count", my_count_function)
	ctx.Global().Set("count_args", cnt)

	res, _ = ctx.Eval(`
            // Now we can call that function in JS
            count_args(1,2,3,4,5)
        `, "compute2.js")

	fmt.Println("count_args(1,2,3,4,5) =", res.String())

	_, err := ctx.Eval(`
            // Sometimes there's a mistake in your js code:
            functin broken(a,b) { return a+b; }
        `, "ooops.js")
	fmt.Println("Err:", err) // <-- get nice error messages

	// output:
	// add(3,4) = 7
	// count_args(1,2,3,4,5) = 5
	// Err: Uncaught exception: SyntaxError: Unexpected identifier
	// at ooops.js:3:20
	//               functin broken(a,b) { return a+b; }
	//                       ^^^^^^
	// Stack trace: SyntaxError: Unexpected identifier
}

func ExampleContext_Create() {
	ctx := v8.NewIsolate().NewContext()

	type Info struct{ Name, Email string }
	fn := func(in v8.CallbackArgs) (*v8.Value, error) {
		return in.Context.Create("yay!")
	}
	var v8val *v8.Value = ctx.Bind("yay_func", fn)

	val, err := ctx.Create(map[string]interface{}{
		"num":    3.7,
		"str":    "simple string",
		"bool":   true,
		"struct": Info{"foo", "bar"},
		"list":   []int{1, 2, 3},
		"func":   fn,    // Callback functions are automatically bound
		"value":  v8val, // Can also include any *v8.Value such as explicitly bound callbacks.
	})

	_, _ = val, err // check errors & use val
}

func ExampleSnapshot() {
	snapshot := v8.CreateSnapshot(`
        // Concantenate all the scripts you want at startup, e.g. lodash, etc.
        _ = { map: function() { /* ... */ }, etc: "etc, etc..." };
        // Setup my per-context global state:
        myGlobalState = {
            init: function() { this.initialized = true; },
            foo: 3,
        };
        // Run some functions:
        myGlobalState.init();
    `)
	iso := v8.NewIsolateWithSnapshot(snapshot)

	// Create a context with the state from the snapshot:
	ctx1 := iso.NewContext()
	fmt.Println("Context 1:")
	val, _ := ctx1.Eval("myGlobalState.foo = 37; myGlobalState.initialized", "")
	fmt.Println("myGlobalState.initialized:", val)
	val, _ = ctx1.Eval("myGlobalState.foo", "")
	fmt.Println("myGlobalState.foo:", val)

	// In the second context, the global state is reset to the state at the
	// snapshot:
	ctx2 := iso.NewContext()
	fmt.Println("Context 2:")
	val, _ = ctx2.Eval("myGlobalState.foo", "")
	fmt.Println("myGlobalState.foo:", val)

	// Output:
	// Context 1:
	// myGlobalState.initialized: true
	// myGlobalState.foo: 37
	// Context 2:
	// myGlobalState.foo: 3
}
