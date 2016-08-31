package v8

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunSimpleJS(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`
		var a = 10;
		var b = 20;
		var c = a+b;
		c;
	`, "test.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	if str := res.String(); str != "30" {
		t.Errorf("Expected 30, got %q", str)
	}
}

func TestErrorRunningInvalidJs(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	res, err := ctx.Eval(`kajsdfa91j23e`, "junk.js")
	if err == nil {
		t.Errorf("Expected error, but got result: %v", res)
	}
}

func TestJsReturnStringWithEmbeddedNulls(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`"foo\000bar"`, "test.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	if str := res.String(); str != "foo\000bar" {
		t.Errorf("Expected 'foo\\000bar', got %q", str)
	}
}

func TestJsReturnUndefined(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(``, "undefined.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	if str := res.String(); str != "undefined" {
		t.Errorf("Expected 'undefined', got %q", str)
	}
}

func TestJsThrowString(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`throw 'badness'`, "my_file.js")
	if err == nil {
		t.Fatalf("It worked but it wasn't supposed to: %v", res.String())
	}
	match, _ := regexp.MatchString("Uncaught exception: badness", err.Error())
	if !match {
		t.Error("Expected 'Uncaught exception: badness', got: ", err.Error())
	}
}

func TestJsThrowError(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`throw new Error('ooopsie')`, "my_file.js")
	if err == nil {
		t.Fatalf("It worked but it wasn't supposed to: %v", res.String())
	}
	match, _ := regexp.MatchString("Uncaught exception: Error: ooopsie", err.Error())
	if !match {
		t.Error("Expected 'Uncaught exception: Error: ooopsie', got: ", err.Error())
	}
}

func TestReadFieldFromObject(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`({foo:"bar"})`, "my_file.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	val, err := res.Get("foo")
	if err != nil {
		t.Fatalf("Error trying to get field: %v", err)
	}
	if str := val.String(); str != "bar" {
		t.Errorf("Expected 'bar', got %q", str)
	}
}

func TestReadAndWriteIndexFromArray(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	val, err := ctx.Create([]int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}

	v, err := val.GetIndex(1)
	if err != nil {
		t.Fatal(err)
	} else if str := v.String(); str != "2" {
		t.Errorf("Wrong value, expected 2, got %s", str)
	}

	v2, err := val.GetIndex(17)
	if err != nil {
		t.Fatal(err)
	} else if str := v2.String(); str != "undefined" {
		t.Errorf("Expected undefined, got %s", str)
	}

	val.SetIndex(17, v)
	v2, err = val.GetIndex(17)
	if err != nil {
		t.Fatal(err)
	} else if str := v2.String(); str != "2" {
		t.Errorf("Expected 2, got %s", str)
	}
}

func TestReadFieldFromNonObjectFails(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`17`, "my_file.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	val, err := res.Get("foo")
	if err == nil {
		t.Fatalf("Missing error trying to get field, got %v", val)
	}
	val, err = res.GetIndex(3)
	if err == nil {
		t.Fatalf("Missing error trying to get field, got %v", val)
	}
}

func TestReadFieldFromGlobal(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	_, err := ctx.Eval(`foo = "bar";`, "my_file.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	val, err := ctx.Global().Get("foo")
	if err != nil {
		t.Fatalf("Error trying to get field: %v", err)
	}
	if str := val.String(); str != "bar" {
		t.Errorf("Expected 'bar', got %q", str)
	}
}

func TestSetField(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	three, err := ctx.Eval(`(3)`, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := ctx.Global().Set("foo", three); err != nil {
		t.Fatal(err)
	}

	res, err := ctx.Eval(`foo`, "")
	if err != nil {
		t.Fatal(err)
	}
	if str := res.String(); str != "3" {
		t.Errorf("Expected 3, got %q", str)
	}
}

func TestRunningCodeInContextAfterThrowingError(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	_, err := ctx.Eval(`
		function fail(a,b) {
			this.c = a+b;
			throw "some failure";
		}
		function work(a,b) {
			this.c = a+b+2;
		}
		x = new fail(3,5);`, "file1.js")
	if err == nil {
		t.Fatal("Expected an exception.")
	}

	res, err := ctx.Eval(`y = new work(3,6); y.c`, "file2.js")
	if err != nil {
		t.Fatal("Expected it to work, but got:", err)
	}

	if str := res.String(); str != "11" {
		t.Errorf("Expected 11, got: %q", str)
	}
}

func TestManyContextsThrowingErrors(t *testing.T) {
	t.Parallel()

	prog := `
		function work(N, depth, fail) {
			if (depth == 0) { return 1; }
			var sum = 0;
			for (i = 0; i < N; i++) { sum *= work(N, depth-1); }
			if (fail) {
				throw "Failed";
			}
			return sum;
		}`

	const N = 100 // num parallel contexts
	runtime.GOMAXPROCS(N)

	var done sync.WaitGroup

	iso := NewIsolate()

	done.Add(N)
	for i := 0; i < N; i++ {
		ctx := iso.NewContext()

		ctx.Eval(prog, "prog.js")
		go func(ctx *Context, i int) {
			cmd := fmt.Sprintf(`work(10000,100,%v)`, i%5 == 0)
			ctx.Eval(cmd, "<inline>")
			ctx.Eval(cmd, "<inline>")
			ctx.Eval(cmd, "<inline>")
			done.Done()
		}(ctx, i)
	}
	done.Wait()
}

func TestErrorsInNativeCode(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	_, err := ctx.Eval(`[].map(undefined);`, "map_undef.js")
	if err == nil {
		t.Fatal("Expected error.")
	}
	t.Log("Got expected error: ", err)
}

func TestCallFunctionWithExplicitThis(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	this, _ := ctx.Eval(`(function(){ this.z = 3; return this; })()`, "")
	add, _ := ctx.Eval(`((x,y)=>(x+y+this.z))`, "")
	one, _ := ctx.Eval(`1`, "")
	two, _ := ctx.Eval(`2`, "")
	res, err := add.Call(this, one, two)
	if err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "6" {
		t.Errorf("Expected 6, got %q", str)
	}
}

func TestCallFunctionWithGlobalScope(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	ctx.Eval(`z = 4`, "")
	add, _ := ctx.Eval(`((x,y)=>(x+y+this.z))`, "")
	one, _ := ctx.Eval(`1`, "")
	two, _ := ctx.Eval(`2`, "")
	res, err := add.Call(nil, one, two)
	if err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "7" {
		t.Errorf("Expected 7, got %q", str)
	}
}

func TestCallFunctionFailsOnNonFunction(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	ob, _ := ctx.Eval(`({x:3})`, "")
	res, err := ob.Call(nil)
	if err == nil {
		t.Fatalf("Expected err, but got %v", res)
	} else if err.Error() != "Not a function" {
		t.Errorf("Wrong error message: %q", err)
	}
}

func TestBind(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	var expectedLoc Loc

	getLastCb := func(in CallbackArgs) (*Value, error) {
		if in.Caller != expectedLoc {
			t.Errorf("Wrong source location: %#v", in.Caller)
		}
		t.Logf("Args: %s", in.Args)
		return in.Args[len(in.Args)-1], nil
	}

	getLast := ctx.Bind("foo", getLastCb)
	ctx.Global().Set("last", getLast)

	expectedLoc = Loc{"doit", "somefile.js", 3, 11}
	res, err := ctx.Eval(`
		function doit() {
			return last(1,2,3);
		}
		doit()
	`, "somefile.js")
	if err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "3" {
		t.Errorf("Expected 3, got %q", str)
	}

	expectedLoc = Loc{"", "", 0, 0} // empty when called directly from Go
	abc, _ := ctx.Eval("'abc'", "unused_filename.js")
	xyz, _ := ctx.Eval("'xyz'", "unused_filename.js")
	res, err = getLast.Call(nil, res, abc, xyz)
	if err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "xyz" {
		t.Errorf("Expected xyz, got %q", str)
	}
}

func TestBindReturnsError(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	fails := ctx.Bind("fails", func(CallbackArgs) (*Value, error) {
		return nil, errors.New("borked")
	})
	res, err := fails.Call(nil)
	if err == nil {
		t.Fatalf("Expected error, but got %q instead", res)
	} else if !strings.HasPrefix(err.Error(), "Uncaught exception: Error: borked") {
		t.Errorf("Wrong error message: %q", err)
	}
}

func TestBindPanics(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	panic := ctx.Bind("panic", func(CallbackArgs) (*Value, error) { panic("aaaah!!") })
	ctx.Global().Set("panic", panic)
	res, err := ctx.Eval(`panic();`, "esplode.js")
	if err == nil {
		t.Error("Expected error, got ", res)
	} else if matched, _ := regexp.MatchString("panic.*aaaah!!", err.Error()); !matched {
		t.Errorf("Error should mention a panic and 'aaaah!!', but doesn't: %v", err)
	}
}

func TestBindName(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	xyz := ctx.Bind("xyz", func(CallbackArgs) (*Value, error) { return nil, nil })
	if str := xyz.String(); str != "function xyz() { [native code] }" {
		t.Errorf("Wrong function signature: %q", str)
	}
}

func TestBindNilReturn(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	xyz := ctx.Bind("xyz", func(CallbackArgs) (*Value, error) { return nil, nil })
	res, err := xyz.Call(nil)
	if err != nil {
		t.Error(err)
	}
	if str := res.String(); str != "undefined" {
		t.Errorf("Expected undefined, got %q", res)
	}
}

func TestTerminate(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	waitUntilRunning := make(chan bool)
	notify := ctx.Bind("notify", func(CallbackArgs) (*Value, error) {
		waitUntilRunning <- true
		return nil, nil
	})
	ctx.Global().Set("notify", notify)

	go func() {
		<-waitUntilRunning
		ctx.Terminate()
	}()

	done := make(chan bool)

	go func() {
		res, err := ctx.Eval(`
			notify();
			while(1) {}
		`, "test.js")

		if err == nil {
			t.Error("Expected an error, but got result: ", res)
		}

		done <- true
	}()

	select {
	case <-done:
		// yay, it worked!
	case <-time.After(time.Second):
		t.Fatal("Terminate didn't terminate :/")
	}
}

func TestSnapshot(t *testing.T) {
	t.Parallel()
	snapshot := CreateSnapshot("zzz='hi there!';")
	ctx := NewIsolateWithSnapshot(snapshot).NewContext()

	res, err := ctx.Eval(`zzz`, "script.js")
	if err != nil {
		t.Fatal(err)
	}
	if str := res.String(); str != "hi there!" {
		t.Errorf("Expected 'hi there!' got %s", str)
	}
}

func TestEs6Destructuring(t *testing.T) {
	if Version.Major < 5 {
		t.Skip("V8 versions before 5.* don't support destructuring.")
	}

	t.Parallel()
	ctx := NewIsolate().NewContext()

	bar, err := ctx.Eval(`
		const f = (n) => ({foo:n, bar:n+1});
		var {foo, bar} = f(5);
		bar
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if str := bar.String(); str != "6" {
		t.Errorf("Expected 6, got %q", str)
	}
}

func TestJsonExport(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	var json_stringify *Value
	if json, err := ctx.Global().Get("JSON"); err != nil {
		t.Fatal(err)
	} else if json_stringify, err = json.Get("stringify"); err != nil {
		t.Fatal(err)
	}

	some_result, err := ctx.Eval("(() => ({a:3,b:'xyz',c:true}))()", "test.js")
	if err != nil {
		t.Fatal(err)
	}

	res, err := json_stringify.Call(json_stringify, some_result)
	if err != nil {
		t.Fatal(err)
	}

	if str := res.String(); str != `{"a":3,"b":"xyz","c":true}` {
		t.Errorf("Wrong JSON result, got: %s", str)
	}
}

func TestValueReleaseMoreThanOnceIsOk(t *testing.T) {
	t.Parallel()
	iso := NewIsolate()
	ctx := iso.NewContext()

	res, err := ctx.Eval("5", "test.js")
	if err != nil {
		t.Fatal(err)
	}
	res.release()
	res.release()
	res.release()
	res.release()

	ctx.release()
	iso.release()
	ctx.release()
	ctx.release()
	iso.release()
	iso.release()
}

func TestSharingValuesAmongContextsInAnIsolate(t *testing.T) {
	t.Parallel()
	iso := NewIsolate()
	ctx1, ctx2 := iso.NewContext(), iso.NewContext()

	//  Create a value in ctx1
	foo, err := ctx1.Eval(`foo = {x:6,y:true,z:"asdf"}; foo`, "ctx1.js")
	if err != nil {
		t.Fatal(err)
	}

	// Set that value into ctx2
	err = ctx2.Global().Set("bar", foo)
	if err != nil {
		t.Fatal(err)
	}
	// ...and verify that it has the same value.
	res, err := ctx2.Eval(`bar.z`, "ctx2.js")
	if err != nil {
		t.Fatal(err)
	}
	if str := res.String(); str != "asdf" {
		t.Errorf("Expected 'asdf', got %q", str)
	}

	// Now modify that value in ctx2
	_, err = ctx2.Eval("bar.z = 'xyz';", "ctx2b.js")
	if err != nil {
		t.Fatal(err)
	}

	// ...and verify that it got changed in ctx1 as well!
	res, err = ctx1.Eval("foo.z", "ctx1b.js")
	if err != nil {
		t.Fatal(err)
	}
	if str := res.String(); str != "xyz" {
		t.Errorf("Expected 'xyz', got %q", str)
	}
}

func TestCreateSimple(t *testing.T) {
	t.Parallel()
	iso := NewIsolate()
	ctx := iso.NewContext()

	callback := func(CallbackArgs) (*Value, error) { return nil, nil }

	var testcases = []struct {
		val interface{}
		str string
	}{
		{nil, "undefined"},
		{3, "3"},
		{3.7, "3.7"},
		{true, "true"},
		{"asdf", "asdf"},
		{callback, "function v8.TestCreateSimple.func1() { [native code] }"},
		{map[string]int{"foo": 1, "bar": 2}, "[object Object]"},
		{struct {
			Foo int
			Bar bool
		}{3, true}, "[object Object]"},
		{[]interface{}{1, true, "three"}, "1,true,three"},
	}

	for i, test := range testcases {
		val, err := ctx.Create(test.val)
		if err != nil {
			t.Errorf("%d: Failed to create %#v: %v", i, test, err)
			continue
		}
		if str := val.String(); str != test.str {
			t.Errorf("Expected %q, got %q", test.str, str)
		}
	}
}

func TestCreateComplex(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	fn := func(CallbackArgs) (*Value, error) { return ctx.Create("abc") }
	type Struct struct {
		Val    string
		secret bool
		Sub    interface{}
	}

	zzz1 := &Struct{Val: "BOOM!"}
	zzz2 := &zzz1
	zzz3 := &zzz2
	zzz4 := &zzz3
	zzz5 := &zzz4 // zzz5 is a *****Struct.  Make sure pointers work!

	val, err := ctx.Create([]Struct{
		{"asdf", false, nil},
		{"foo", true, map[string]interface{}{
			"num":  123.123,
			"fn":   fn,
			"fn2":  ctx.Bind("fn2", fn),
			"list": []float64{1, 2, 3},
		}},
		{"*****Struct", false, zzz5},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx.Global().Set("mega", val)

	if res, err := ctx.Eval(`mega[1].Sub.fn2()`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "abc" {
		t.Errorf("Expected abc, got %q", str)
	}

	if res, err := ctx.Eval(`mega[1].Sub.fn()`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "abc" {
		t.Errorf("Expected abc, got %q", str)
	}

	if res, err := ctx.Eval(`mega[1].secret`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "undefined" {
		t.Errorf("Expected undefined trying to access non-existent field 'secret', but got %q", str)
	}

	if res, err := ctx.Eval(`mega[2].Sub.Val`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "BOOM!" {
		t.Errorf("Expected 'BOOM1', but got %q", str)
	}
}

func TestCreateJsonTags(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	type A struct {
		Embedded    string `json:"embedded"`
		AlsoIngored string `json:"-"`
	}
	type Nil struct{ Missing string }
	type B struct {
		*A
		*Nil
		Ignored     string `json:"-"`
		Renamed     string `json:"foo"`
		DefaultName string `json:",omitempty"`
		Bar         string
	}

	var x = B{&A{"a", "x"}, nil, "y", "b", "c", "d"}
	val, err := ctx.Create(x)
	if err != nil {
		t.Fatal(err)
	}

	const expected = `{"embedded":"a","foo":"b","DefaultName":"c","Bar":"d"}`
	if data, err := json.Marshal(val); err != nil {
		t.Fatal(err)
	} else if string(data) != expected {
		t.Errorf("Incorrect object:\nExp: %s\nGot: %s", expected, data)
	}
}

func TestParseJson(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	val, err := ctx.ParseJson(`{"foo":"bar","bar":3}`)
	if err != nil {
		t.Fatal(err)
	}
	if res, err := val.Get("foo"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "bar" {
		t.Errorf("Expected 'bar', got %q", str)
	}

	// Make sure it fails if the data is not actually json.
	val, err = ctx.ParseJson(`this is not json`)
	if err == nil {
		t.Errorf("Expected an error, but got %s", val)
	}
}

func TestJsonMarshal(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	val, err := ctx.Eval(`(()=>({
		blah: 3,
		muck: true,
		label: "lala",
		missing: () => ( "functions get dropped" )
	}))()`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"blah":3,"muck":true,"label":"lala"}`
	if string(data) != expected {
		t.Errorf("Expected: %q\nGot     : %q", expected, string(data))
	}
}

func TestCallbackProvideCorrectContext(t *testing.T) {
	t.Parallel()

	// greet is a generate callback handler that is not associated with a
	// particular context -- it uses the provided context to create a value
	// to return, even when used from different isolates.
	greet := func(in CallbackArgs) (*Value, error) {
		return in.Context.Create("Hello " + in.Arg(0).String())
	}

	ctx1, ctx2 := NewIsolate().NewContext(), NewIsolate().NewContext()
	ctx1.Global().Set("greet", ctx1.Bind("greet", greet))
	ctx2.Global().Set("greet", ctx2.Bind("greet", greet))

	alice, err1 := ctx1.Eval("greet('Alice')", "ctx1.js")
	if err1 != nil {
		t.Errorf("Context 1 failed: %v", err1)
	} else if str := alice.String(); str != "Hello Alice" {
		t.Errorf("Bad result: %q", str)
	}

	bob, err2 := ctx2.Eval("greet('Bob')", "ctx2.js")
	if err2 != nil {
		t.Errorf("Context 2 failed: %v", err2)
	} else if str := bob.String(); str != "Hello Bob" {
		t.Errorf("Bad result: %q", str)
	}
}
