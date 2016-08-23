package v8console_test

import (
	"fmt"
	"os"

	"github.com/augustoroman/v8"
	"github.com/augustoroman/v8/v8console"
)

func ExampleFlushSnapshotAndInject() {
	const myJsCode = `
        // Typically this will be an auto-generated js bundle file.
        function require() {} // fake stub
        var when = require('when');
        var _ = require('lodash');
        function renderPage(name) { return "<html><body>Hi " + name + "!"; }
        console.warn('snapshot initialization');
    `
	snapshot := v8.CreateSnapshot(v8console.WrapForSnapshot(myJsCode))
	ctx := v8.NewIsolateWithSnapshot(snapshot).NewContext()
	console := v8console.Config{"console> ", os.Stdout, os.Stdout, false}
	if exception := v8console.FlushSnapshotAndInject(ctx, console); exception != nil {
		panic(fmt.Errorf("Panic during snapshot creation: %v", exception.String()))
	}
	_, err := ctx.Eval(`console.warn('after snapshot');`, `somefile.js`)
	if err != nil {
		panic(err)
	}

	// Output:
	// console> [<embedded>:8] snapshot initialization
	// console> [somefile.js:1] after snapshot
}

func ExampleConfig() {
	ctx := v8.NewIsolate().NewContext()
	v8console.Config{"> ", os.Stdout, os.Stdout, false}.Inject(ctx)
	ctx.Eval(`
        console.log('hi there');
        console.info('info 4 u');
        console.warn("Where's mah bucket?");
        console.error("Oh noes!");
    `, "filename.js")
	// You can also update the console:
	v8console.Config{":-> ", os.Stdout, os.Stdout, false}.Inject(ctx)
	ctx.Eval(`console.log("I'm so happy");`, "file2.js")

	// Output:
	// > hi there
	// > info 4 u
	// > [filename.js:4] Where's mah bucket?
	// > [filename.js:5] Oh noes!
	// :-> I'm so happy
}
