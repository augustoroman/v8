package v8

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
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
	if num := res.Int64(); num != 30 {
		t.Errorf("Expected 30, got %v", res)
	}
}

func TestBoolConversion(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	testcases := []struct {
		js       string
		expected bool
		isBool   bool
	}{
		// These are the only values that are KindBoolean. Everything else below is
		// implicitly converted.
		{`true`, true, true},
		{`false`, false, true},

		// Super confusing in JS:
		//   !!(new Boolean(false)) == true
		//   !!(new Boolean(true)) == true
		// That's because a non-undefined non-null Object in JS is 'true'.
		// Also, neither of these are actually Boolean kinds -- they are
		// BooleanObject, though.
		{`new Boolean(true)`, true, false},
		{`new Boolean(false)`, true, false},
		{`undefined`, false, false},
		{`null`, false, false},
		{`[]`, true, false},
		{`[1]`, true, false},
		{`7`, true, false},
		{`"xyz"`, true, false},
		{`(() => 3)`, true, false},
	}

	for i, test := range testcases {
		res, err := ctx.Eval(test.js, "test.js")
		if err != nil {
			t.Errorf("%d %#q: Failed to run js: %v", i, test.js, err)
		} else if b := res.Bool(); b != test.expected {
			t.Errorf("%d %#q: Expected bool of %v, but got %v", i, test.js, test.expected, b)
		} else if res.IsKind(KindBoolean) != test.isBool {
			t.Errorf("%d %#q: Expected this to be a bool kind, but it's %v", i, test.js, res.kindMask)
		}
	}
}

func TestJsRegex(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	re, err := ctx.Eval(`/foo.*bar/`, "test.js")
	if err != nil {
		t.Fatal(err)
	}
	if re.String() != `/foo.*bar/` {
		t.Errorf("Bad stringification of regex: %#q", re)
	}
	if !re.IsKind(KindRegExp) {
		t.Errorf("Wrong kind for regex: %v", re.kindMask)
	}
}

func TestNumberConversions(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	res, err := ctx.Eval(`13`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	if !res.IsKind(KindNumber) {
		t.Errorf("Expected %q to be a number kind, but it's not: %q", res, res.kindMask)
	}
	if res.IsKind(KindFunction) {
		t.Errorf("Expected %q to NOT be a function kind, but it is: %q", res, res.kindMask)
	}

	if f64 := res.Float64(); f64 != 13.0 {
		t.Errorf("Expected %q to eq 13.0, but got %f", res, f64)
	}

	if i64 := res.Int64(); i64 != 13 {
		t.Errorf("Expected %q to eq 13.0, but got %d", res, i64)
	}
}

func TestNumberConversionsFailForNonNumbers(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	res, err := ctx.Eval(`undefined`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	if res.IsKind(KindNumber) {
		t.Errorf("Expected %q to NOT be a number kind, but it is: %q", res, res.kindMask)
	}

	if f64 := res.Float64(); !math.IsNaN(f64) {
		t.Errorf("Expected %q to be NaN, but got %f", res, f64)
	}

	if i64 := res.Int64(); i64 != 0 {
		t.Errorf("Expected %q to eq 0, but got %d", res, i64)
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

func TestValueString(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	testcases := []struct{ jsCode, toString string }{
		// primitives:
		{`"some string"`, `some string`},
		{`5`, `5`},
		{`5.123`, `5.123`},
		{`true`, `true`},
		{`false`, `false`},
		{`null`, `null`},
		{`undefined`, `undefined`},
		// more complicated objects:
		{`(function x() { return 1 + 2; })`, `function x() { return 1 + 2; }`},
		{`([1,2,3])`, `1,2,3`},
		{`({x: 5})`, `[object Object]`},
		// basically a primitive, but an interesting case still:
		{`JSON.stringify({x: 5})`, `{"x":5}`},
	}

	for i, test := range testcases {
		res, err := ctx.Eval(test.jsCode, "test.js")
		if err != nil {
			t.Fatalf("Case %d: Error evaluating javascript %#q, err: %v",
				i, test.jsCode, err)
		}
		if res.String() != test.toString {
			t.Errorf("Case %d: Got %#q, expected %#q from running js %#q",
				i, res.String(), test.toString, test.jsCode)
		}
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
	if b := res.Bytes(); b != nil {
		t.Errorf("Expected failure to map to bytes but got byte array of length %d", len(b))
	}
}

func TestJsReturnArrayBuffer(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	res, err := ctx.Eval(`new ArrayBuffer(5)`, "undefined.js")
	if err != nil {
		t.Fatalf("Error evaluating javascript, err: %v", err)
	}
	b := res.Bytes()
	if b == nil {
		t.Errorf("Expected non-nil byte array but got nil buffer")
	}
	if len(b) != 5 {
		t.Errorf("Expected byte array of length 5 but got %d", len(b))
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

func TestReadAndWriteIndexFromArrayBuffer(t *testing.T) {
	t.Parallel()

	ctx := NewIsolate().NewContext()
	val, err := ctx.Create(struct {
		Data []byte `v8:"arraybuffer"`
	}{[]byte{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}

	data, err := val.Get("Data")
	if err != nil {
		t.Fatal(err)
	}

	v, err := data.GetIndex(1)
	if err != nil {
		t.Fatal(err)
	} else if num := v.Int64(); num != 2 {
		t.Errorf("Wrong value, expected 2, got %v (%v)", num, v)
	}

	v2, err := data.GetIndex(17)
	if err != nil {
		t.Fatal(err)
	} else if str := v2.String(); str != "undefined" {
		t.Errorf("Expected undefined, got %s", str)
	}

	v3, err := data.GetIndex(2)
	if err != nil {
		t.Fatal(err)
	} else if num := v3.Int64(); num != 3 {
		t.Errorf("Expected undefined, got %v (%v)", num, v3)
	}

	data.SetIndex(2, v)
	v2, err = data.GetIndex(2)
	if err != nil {
		t.Fatal(err)
	} else if num := v2.Int64(); num != 2 {
		t.Errorf("Expected 2, got %v (%v)", num, v2)
	}

	largeValue, err := ctx.Create(int(500))
	if err != nil {
		t.Fatal(err)
	}

	// 500 truncates to 500 % 256 == 244
	data.SetIndex(2, largeValue)
	v4, err := data.GetIndex(2)
	if err != nil {
		t.Fatal(err)
	} else if num := v4.Int64(); num != 244 {
		t.Errorf("Expected 244, got %v (%v)", num, v4)
	}

	negativeValue, err := ctx.Create(int(-55))
	if err != nil {
		t.Fatal(err)
	}

	// -55 "truncates" to -55 % 256 == 201
	data.SetIndex(2, negativeValue)
	v5, err := data.GetIndex(2)
	if err != nil {
		t.Fatal(err)
	} else if num := v5.Int64(); num != 201 {
		t.Errorf("Expected 201, got %v (%v)", num, v5)
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
	} else if num := v.Int64(); num != 2 {
		t.Errorf("Wrong value, expected 2, got %v (%v)", num, v)
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
	} else if num := v2.Int64(); num != 2 {
		t.Errorf("Expected 2, got %v (%v)", num, v2)
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
	if num := res.Int64(); num != 3 {
		t.Errorf("Expected 3, got %v (%v)", num, res)
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

	if num := res.Int64(); num != 11 {
		t.Errorf("Expected 11, got: %v (%v)", num, res)
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
	} else if num := res.Int64(); num != 6 {
		t.Errorf("Expected 6, got %v (%v)", num, res)
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
	} else if num := res.Int64(); num != 7 {
		t.Errorf("Expected 7, got %v (%v)", num, res)
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

func TestNewFunction(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	cons, _ := ctx.Eval(`(function(){ this.x = 1; })`, "")
	obj, err := cons.New()
	if err != nil {
		t.Fatal(err)
	}
	res, err := obj.Get("x")
	if err != nil {
		t.Fatal(err)
	} else if num := res.Int64(); num != 1 {
		t.Errorf("Expected 1, got %v (%v)", num, res)
	}
}

func TestNewFunctionThrows(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	cons, _ := ctx.Eval(`(function(){ throw "oops"; })`, "")
	obj, err := cons.New()
	if err == nil {
		t.Fatalf("Expected err, but got %v", obj)
	} else if !strings.HasPrefix(err.Error(), "Uncaught exception: oops") {
		t.Errorf("Wrong error message: %q", err)
	}
}

func TestNewFunctionWithArgs(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	cons, _ := ctx.Eval(`(function(x, y){ this.x = x + y; })`, "")
	one, _ := ctx.Eval(`1`, "")
	two, _ := ctx.Eval(`2`, "")
	obj, err := cons.New(one, two)
	if err != nil {
		t.Fatal(err)
	}
	res, err := obj.Get("x")
	if err != nil {
		t.Fatal(err)
	} else if num := res.Int64(); num != 3 {
		t.Errorf("Expected 3, got %v (%v)", num, res)
	}
}

func TestNewFunctionFailsOnNonFunction(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	ob, _ := ctx.Eval(`({x:3})`, "")
	res, err := ob.New()
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
	} else if num := res.Int64(); num != 3 {
		t.Errorf("Expected 3, got %v (%v)", num, res)
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

func TestSnapshotBadJs(t *testing.T) {
	t.Parallel()
	snapshot := CreateSnapshot("This isn't yo mama's snapshot!")

	if snapshot.data.ptr != nil {
		t.Error("Expected nil ptr")
	}

	ctx := NewIsolateWithSnapshot(snapshot).NewContext()

	_, err := ctx.Eval(`zzz`, "script.js")
	if err == nil {
		t.Fatal("Expected error because zzz should be undefined.")
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
	if num := bar.Int64(); num != 6 {
		t.Errorf("Expected 6, got %v (%v)", num, bar)
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

	tm := time.Date(2018, 5, 8, 3, 4, 5, 17, time.Local)

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
		{tm, tm.Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")},
		{&tm, tm.Format("Mon Jan 02 2006 15:04:05 GMT-0700 (MST)")},
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

	fn2 := ctx.Bind("fn2", fn)

	val, err := ctx.Create([]Struct{
		{"asdf", false, nil},
		{"foo", true, map[string]interface{}{
			"num":    123.123,
			"fn":     fn,
			"fn2":    fn2,
			"list":   []float64{1, 2, 3},
			"valArr": []*Value{fn2},
		}},
		{"*****Struct", false, zzz5},
		{"bufbuf", false, struct {
			Data []byte `v8:"arraybuffer"`
		}{[]byte{1, 2, 3, 4}}},
		{"emptybuf", false, struct {
			Data []byte `v8:"arraybuffer"`
		}{[]byte{}}},
		{"structWithValue", false, struct{ *Value }{fn2}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if fn2.ptr == nil {
		t.Error("Create should not release *Values allocated prior to the call.")
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

	if res, err := ctx.Eval(`mega[3].Sub.Data.byteLength`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "4" {
		t.Errorf("Expected array buffer length of '4', but got %q", str)
	}

	if res, err := ctx.Eval(`new Uint8Array(mega[3].Sub.Data)[2]`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "3" {
		t.Errorf("Expected array buffer value at index 2 of '3', but got %q", str)
	}

	if res, err := ctx.Eval(`mega[4].Sub.Data.byteLength`, "test.js"); err != nil {
		t.Fatal(err)
	} else if str := res.String(); str != "0" {
		t.Errorf("Expected empty array buffer length of '0', but got %q", str)
	}
}

func TestJsCreateArrayBufferRoundtrip(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	val, err := ctx.Create(struct {
		Data []byte `v8:"arraybuffer"`
	}{[]byte{1, 2, 3, 4}})
	if err != nil {
		t.Fatal(err)
	}

	ctx.Global().Set("buf", val)

	if _, err := ctx.Eval(`
		var view = new Uint8Array(buf.Data)
		view[3] = view[0] + view[1] + view[2]
		`, "test.js"); err != nil {
		t.Fatal(err)
	}

	data, err := val.Get("Data")
	if err != nil {
		t.Fatal(err)
	}

	v1, err := data.GetIndex(0)
	if err != nil {
		t.Fatal(err)
	} else if str := v1.String(); str != "1" {
		t.Errorf("Expected first value of '1' but got %q", str)
	}

	v2, err := data.GetIndex(3)
	if err != nil {
		t.Fatal(err)
	} else if str := v2.String(); str != "6" {
		t.Errorf("Expected fourth value of '6' but got %q", str)
	}

	err = data.SetIndex(2, v1)
	if err != nil {
		t.Fatal(err)
	}

	v3, err := data.GetIndex(2)
	if err != nil {
		t.Fatal(err)
	} else if str := v3.String(); str != "1" {
		t.Errorf("Expected third value of '1' but got %q", str)
	}

	bytes := data.Bytes()
	if !reflect.DeepEqual(bytes, []byte{1, 2, 1, 6}) {
		t.Errorf("Expected byte array [1,2,1,6] but got %q", bytes)
	}

	// Out of range
	err = data.SetIndex(7, v1)
	if err == nil {
		t.Errorf("Expected error assigning out of range of array buffer")
	}
}

func TestTypedArrayBuffers(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	uint8Array, err := ctx.Eval(`
		new Uint8Array(4).fill(4, 1, 3) // taken from a MDN example
	`, "test.js")
	if err != nil {
		t.Fatal(err)
	}

	bytes := uint8Array.Bytes()
	if !reflect.DeepEqual(bytes, []byte{0, 4, 4, 0}) {
		t.Errorf("Expected byte array [0,4,4,0] but got %q", bytes)
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

func TestCircularReferenceJsonMarshalling(t *testing.T) {
	t.Parallel()

	ctx := NewIsolate().NewContext()
	circ, err := ctx.Eval("var test = {}; test.blah = test", "circular.js")
	if err != nil {
		t.Fatalf("Failed to create object with circular ref: %v", err)
	}
	data, err := circ.MarshalJSON()
	if err == nil {
		t.Fatalf("Expected error marshalling circular ref, but got: `%s`", data)
	} else if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected a circular reference error, but got: %v", err)
	}
}

func TestIsolateFinalizer(t *testing.T) {
	t.Parallel()
	iso := NewIsolate()

	fin := make(chan bool)
	// Reset the finalizer so we test if it is working
	runtime.SetFinalizer(iso, nil)
	runtime.SetFinalizer(iso, func(iso *Isolate) {
		close(fin)
		iso.release()
	})
	iso = nil

	if !runGcUntilReceivedOrTimedOut(fin, 4*time.Second) {
		t.Fatal("finalizer of iso didn't run, no context is associated with the iso.")
	}

	iso = NewIsolate()
	iso.NewContext()

	fin = make(chan bool)
	// Reset the finalizer so we test if it is working
	runtime.SetFinalizer(iso, nil)
	runtime.SetFinalizer(iso, func(iso *Isolate) {
		close(fin)
		iso.release()
	})
	iso = nil

	if !runGcUntilReceivedOrTimedOut(fin, 4*time.Second) {
		t.Fatal("finalizer of iso didn't run, iso created one context.")
	}
}

func TestContextFinalizer(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	fin := make(chan bool)
	// Reset the finalizer so we test if it is working
	runtime.SetFinalizer(ctx, nil)
	runtime.SetFinalizer(ctx, func(ctx *Context) {
		close(fin)
		ctx.release()
	})
	ctx = nil

	if !runGcUntilReceivedOrTimedOut(fin, 4*time.Second) {
		t.Fatal("finalizer of ctx didn't run")
	}
}

func TestContextFinalizerWithValues(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	greet := func(in CallbackArgs) (*Value, error) {
		return in.Context.Create("Hello " + in.Arg(0).String())
	}
	ctx.Global().Set("greet", ctx.Bind("greet", greet))
	val, err := ctx.Eval("greet('bob')", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(val.String())

	fin := make(chan bool)
	// Reset the finalizer so we test if it is working
	runtime.SetFinalizer(ctx, nil)
	runtime.SetFinalizer(ctx, func(ctx *Context) {
		close(fin)
		ctx.release()
	})
	ctx = nil

	if !runGcUntilReceivedOrTimedOut(fin, 4*time.Second) {
		t.Fatal("finalizer of ctx didn't run after creating a value")
	}
}

func TestIsolateGetHeapStatistics(t *testing.T) {
	iso := NewIsolate()
	initHeap := iso.GetHeapStatistics()
	if initHeap.TotalHeapSize <= 0 {
		t.Fatalf("expected heap to be more than zero, got: %d\n", initHeap.TotalHeapSize)
	}

	ctx := iso.NewContext()
	for i := 0; i < 10000; i++ {
		ctx.Create(map[string]interface{}{
			"hello": map[string]interface{}{
				"world": []string{"foo", "bar"},
			},
		})
	}

	midHeap := iso.GetHeapStatistics()
	if midHeap.TotalHeapSize <= initHeap.TotalHeapSize {
		t.Fatalf("expected heap to grow after creating context, got: %d\n", midHeap.TotalHeapSize)
	}

	beforeNotifyHeap := iso.GetHeapStatistics()

	iso.SendLowMemoryNotification()

	finalHeap := iso.GetHeapStatistics()
	if finalHeap.TotalHeapSize >= beforeNotifyHeap.TotalHeapSize {
		t.Fatalf("expected heap to reduce after terminating context, got: %d\n", finalHeap.TotalHeapSize)
	}

}

func runGcUntilReceivedOrTimedOut(signal <-chan bool, timeout time.Duration) bool {
	expired := time.After(timeout)
	for {
		select {
		case <-signal:
			return true
		case <-expired:
			return false
		case <-time.After(10 * time.Millisecond):
			runtime.GC()
		}
	}
}

// This is bad, and should be fixed! See https://github.com/augustoroman/v8/issues/21
func TestMicrotasksIgnoreUnhandledPromiseRejection(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()
	var logs []string
	ctx.Global().Set("log", ctx.Bind("log", func(in CallbackArgs) (*Value, error) {
		logs = append(logs, in.Arg(0).String())
		return nil, nil
	}))
	output, err := ctx.Eval(`
		log('start');
		let p = new Promise((_, reject) => { log("reject:'err'"); reject('err'); });
		p.then(v => log('then:'+v));
		log('done');
	`, `test.js`)

	expectedLogs := []string{
		"start",
		"reject:'err'",
		"done",
	}

	if !reflect.DeepEqual(logs, expectedLogs) {
		t.Errorf("Wrong logs.\nGot: %#q\nExp: %#q", logs, expectedLogs)
	}

	// output should be 'undefined' because log('done') doesn't return anything.
	if output.String() != "undefined" {
		t.Errorf("Unexpected output value: %v", output)
	}

	if err != nil {
		t.Errorf("Expected err to be nil since we ignore unhandled promise rejections. "+
			"In the future, hopefully we'll handle these better -- in fact, maybe err "+
			"is not-nil right now because you fixed that!  Got err = %v", err)
	}
}

func TestValueKind(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	// WASM: This wasm code corresponds to the WAT:
	//   (module
	//      (func $add (param $x i32) (param $y i32) (result i32)
	//          (i32.add (get_local $x) (get_local $y)))
	//      (export "add" $add))
	// That exports an "add(a,b int32) int32" function.
	const wasmCode = `
		new Uint8Array([0,97,115,109,1,0,0,0,1,135,128,128,128,0,1,96,2,127,127,1,127,3,130,128,128,
			128,0,1,0,6,129,128,128,128,0,0,7,135,128,128,128,0,1,3,97,100,100,0,0,10,141,128,128,128,
			0,1,135,128,128,128,0,0,32,0,32,1,106,11])`

	const wasmModule = `new WebAssembly.Module(` + wasmCode + `)`

	toTest := map[string]kindMask{
		`undefined`:                        mask(KindUndefined),
		`null`:                             mask(KindNull),
		`"test"`:                           unionKindString,
		`Symbol("test")`:                   unionKindSymbol,
		`(function(){})`:                   unionKindFunction,
		`[]`:                               unionKindArray,
		`new Object()`:                     mask(KindObject),
		`true`:                             mask(KindBoolean),
		`false`:                            mask(KindBoolean),
		`1`:                                mask(KindNumber, KindInt32, KindUint32),
		`new Date()`:                       unionKindDate,
		`(function(){return arguments})()`: unionKindArgumentsObject,
		`new Boolean`:                      unionKindBooleanObject,
		`new Number`:                       unionKindNumberObject,
		`new String`:                       unionKindStringObject,
		`new Object(Symbol("test"))`:       unionKindSymbolObject,
		`/regexp/`:                         unionKindRegExp,
		`new Promise((res, rjt)=>{})`:      unionKindPromise,
		`new Map()`:                        unionKindMap,
		`new Set()`:                        unionKindSet,
		`new ArrayBuffer(0)`:               unionKindArrayBuffer,
		`new Uint8Array(0)`:                unionKindUint8Array,
		`new Uint8ClampedArray(0)`:         unionKindUint8ClampedArray,
		`new Int8Array(0)`:                 unionKindInt8Array,
		`new Uint16Array(0)`:               unionKindUint16Array,
		`new Int16Array(0)`:                unionKindInt16Array,
		`new Uint32Array(0)`:               unionKindUint32Array,
		`new Int32Array(0)`:                unionKindInt32Array,
		`new Float32Array(0)`:              unionKindFloat32Array,
		`new Float64Array(0)`:              unionKindFloat64Array,
		`new DataView(new ArrayBuffer(0))`: unionKindDataView,
		`new SharedArrayBuffer(0)`:         unionKindSharedArrayBuffer,
		`new Proxy({}, {})`:                unionKindProxy,
		`new WeakMap`:                      unionKindWeakMap,
		`new WeakSet`:                      unionKindWeakSet,
		`(async function(){})`:             unionKindAsyncFunction,
		`(function* (){})`:                 unionKindGeneratorFunction,
		`function* gen(){}; gen()`:         unionKindGeneratorObject,
		`new Map()[Symbol.iterator]()`:     unionKindMapIterator,
		`new Set()[Symbol.iterator]()`:     unionKindSetIterator,
		`new EvalError`:                    unionKindNativeError,
		wasmModule:                         unionKindWebAssemblyCompiledModule,

		// TODO!
		// ``: KindExternal,
	}

	for script, kindMask := range toTest {
		v, err := ctx.Eval(script, "kind_test.js")
		if err != nil {
			t.Errorf("%#q: failed: %v", script, err)
		} else if v.kindMask != kindMask {
			t.Errorf("%#q: expected result to be %q, but got %q", script, kindMask, v.kindMask)
		}
	}
}

func TestDate(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	res, err := ctx.Eval(`new Date("2018-05-08T08:16:46.918Z")`, "date.js")
	if err != nil {
		t.Fatal(err)
	}

	tm, err := res.Date()
	if err != nil {
		t.Error(err)
	} else if tm.UnixNano() != 1525767406918*1e6 {
		t.Errorf("Wrong date: %q", tm)
	}
}

func TestPromise(t *testing.T) {
	t.Parallel()
	ctx := NewIsolate().NewContext()

	// Pending
	v, err := ctx.Eval(`new Promise((resolve, reject)=>{})`, "pending-promise.js")
	if err != nil {
		t.Fatal(err)
	}

	if state, result, err := v.PromiseInfo(); err != nil {
		t.Error(err)
	} else if state != PromiseStatePending {
		t.Errorf("Expected promise to be pending, but got %v", state)
	} else if result != nil {
		t.Errorf("Expected nil result since it's pending, but got %v", result)
	}

	// Resolved
	v, err = ctx.Eval(`new Promise((resolve, reject)=>{resolve(42)})`, "resolved-promise.js")
	if err != nil {
		t.Fatal(err)
	}

	if state, result, err := v.PromiseInfo(); err != nil {
		t.Error(err)
	} else if state != PromiseStateResolved {
		t.Errorf("Expected promise to be resolved, but got %v", state)
	} else if result == nil {
		t.Errorf("Expected a result since it's resolved, but got nil")
	} else if !result.IsKind(KindNumber) {
		t.Errorf("Expected the result to be a number, but it's: %v (%v)", result.kindMask, result)
	} else if result.Int64() != 42 {
		t.Errorf("Expected the result to be 42, but got %v", result)
	}

	// Rejected
	v, err = ctx.Eval(`new Promise((resolve, reject)=>{reject(new Error("nope"))})`, "rejected-promise.js")
	if err != nil {
		t.Fatal(err)
	}

	if state, result, err := v.PromiseInfo(); err != nil {
		t.Error(err)
	} else if state != PromiseStateRejected {
		t.Errorf("Expected promise to be rejected, but got %v", state)
	} else if result == nil {
		t.Errorf("Expected an error result since it's rejected, but got nil")
	} else if !result.IsKind(KindNativeError) {
		t.Errorf("Expected the result to be an error, but it's: %v (%v)", result.kindMask, result)
	} else if result.String() != `Error: nope` {
		t.Errorf("Expected the error message to be 'nope', but got %#q", result)
	}

	// Not a promise
	v, err = ctx.Eval(`new Error('x')`, "not-a-promise.js")
	if err != nil {
		t.Fatal(err)
	}

	if state, result, err := v.PromiseInfo(); err == nil {
		t.Errorf("Expected an error, but got nil and state=%#v result=%#v", state, result)
	}
}

func TestPanicHandling(t *testing.T) {
	// v8 runtime can register its own signal handlers which would interfere
	// with Go's signal handlers which are needed for panic handling
	defer func() {
		if r := recover(); r != nil {
			// if we reach this point, Go's panic mechanism is still intact
			_, ok := r.(runtime.Error)
			if !ok {
				t.Errorf("expected runtime error, actual %v", r)
			}
		}
	}()

	var f *big.Float
	_ = NewIsolate()
	_ = *f
}
