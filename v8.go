package v8

// Reference materials:
//   https://developers.google.com/v8/embed#accessors
//   https://developers.google.com/v8/embed#exceptions
//   https://docs.google.com/document/d/1g8JFi8T_oAE_7uAri7Njtig7fKaPDfotU6huOa1alds/edit
// TODO:
//   Value.Export(v) --> inverse of Context.Create()
//   Proxy objects

// #include <stdlib.h>
// #include <string.h>
// #include "v8_c_bridge.h"
// #cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -std=c++11
// #cgo LDFLAGS: -L${SRCDIR}/libv8 -lv8_base -lv8_libbase -lv8_snapshot -lv8_libsampler -lv8_libplatform -ldl -pthread
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// Callback is the signature for callback functions that are registered with a
// V8 context via Bind(). Never return a Value from a different V8 isolate. A
// return value of nil will return "undefined" to javascript. Returning an
// error will throw an exception. Panics are caught and returned as errors to
// avoid disrupting the cgo stack.
type Callback func(CallbackArgs) (*Value, error)

// CallbackArgs provide the context for handling a javascript callback into go.
// Caller is the script location that javascript is calling from. If the
// function is called directly from Go (e.g. via Call()), then "Caller" will be
// empty. Args are the arguments provided by the JS code.  Context is the V8
// context that initiated the call.
type CallbackArgs struct {
	Caller  Loc
	Args    []*Value
	Context *Context
}

// Arg returns the specified argument or "undefined" if it doesn't exist.
func (c *CallbackArgs) Arg(n int) *Value {
	if n < len(c.Args) && n >= 0 {
		return c.Args[n]
	}
	undef, _ := c.Context.Create(nil)
	return undef
}

// Loc defines a script location.
type Loc struct {
	Funcname, Filename string
	Line, Column       int
}

// Version exposes the compiled-in version of the linked V8 library.  This can
// be used to test for specific javascript functionality support (e.g. ES6
// destructuring isn't supported before major version 5.).
var Version = struct{ Major, Minor, Build, Patch int }{
	Major: int(C.version.Major),
	Minor: int(C.version.Minor),
	Build: int(C.version.Build),
	Patch: int(C.version.Patch),
}

// Ensure that v8 is initialized exactly once on first use.
var v8_init_once sync.Once

// Snapshot contains the stored VM state that can be used to quickly recreate a
// new VM at that particular state.
type Snapshot struct{ data C.StartupData }

func newSnapshot(data C.StartupData) *Snapshot {
	s := &Snapshot{data}
	runtime.SetFinalizer(s, (*Snapshot).release)
	return s
}

func (s *Snapshot) release() {
	if s.data.ptr != nil {
		C.free(unsafe.Pointer(s.data.ptr))
	}
	s.data.ptr = nil
	s.data.len = 0
}

// Export returns the VM state data as a byte slice.
func (s *Snapshot) Export() []byte {
	return []byte(C.GoStringN(s.data.ptr, s.data.len))
}

// RestoreSnapshotFromExport creates a Snapshot from a byte slice that should
// have previous come from Snapshot.Export().
func RestoreSnapshotFromExport(data []byte) *Snapshot {
	str := C.String{
		ptr: (*C.char)(C.malloc(C.size_t(len(data)))),
		len: C.int(len(data)),
	}
	C.memcpy(unsafe.Pointer(str.ptr), unsafe.Pointer(&data[0]), C.size_t(len(data)))
	return newSnapshot(str)
}

// CreateSnapshot creates a new Snapshot after running the supplied JS code.
// Because Snapshots cannot have refences to external code (no Go callbacks),
// all of the initialization code must be pure JS and supplied at once as the
// arg to this function.
func CreateSnapshot(js string) *Snapshot {
	v8_init_once.Do(func() { C.v8_init() })
	js_ptr := C.CString(js)
	defer C.free(unsafe.Pointer(js_ptr))
	return newSnapshot(C.v8_CreateSnapshotDataBlob(js_ptr))
}

// Isolate represents a single-threaded V8 engine instance.  It can run multiple
// independent Contexts and V8 values can be freely shared between the Contexts,
// however only one context will ever execute at a time.
type Isolate struct{ ptr C.IsolatePtr }

// NewIsolate creates a new V8 Isolate.
func NewIsolate() *Isolate {
	v8_init_once.Do(func() { C.v8_init() })
	iso := &Isolate{C.v8_Isolate_New(C.StartupData{ptr: nil, len: 0})}
	runtime.SetFinalizer(iso, (*Isolate).release)
	return iso
}

// NewIsolateWithSnapshot creates a new V8 Isolate using the supplied Snapshot
// to initialize all Contexts created from this Isolate.
func NewIsolateWithSnapshot(s *Snapshot) *Isolate {
	v8_init_once.Do(func() { C.v8_init() })
	iso := &Isolate{C.v8_Isolate_New(s.data)}
	runtime.SetFinalizer(iso, (*Isolate).release)
	return iso
}

// NewContext creates a new, clean V8 Context within this Isolate.
func (i *Isolate) NewContext() *Context {
	ctx := &Context{
		iso:       i,
		ptr:       C.v8_Isolate_NewContext(i.ptr),
		callbacks: map[int]callbackInfo{},
		values:    map[*Value]bool{},
	}

	contextsMutex.Lock()
	nextContextId++
	id := nextContextId
	ctx.id = id
	contexts[id] = ctx
	contextsMutex.Unlock()

	runtime.SetFinalizer(ctx, (*Context).release)

	return ctx
}

// Terminate will interrupt all operation in this Isolate, interrupting any
// Contexts that are executing.  This may be called from any goroutine at any
// time.
func (i *Isolate) Terminate() { C.v8_Isolate_Terminate(i.ptr) }
func (i *Isolate) release()   { C.v8_Isolate_Release(i.ptr); i.ptr = nil }

func (i *Isolate) convertErrorMsg(error_msg C.Error) error {
	if error_msg.ptr == nil {
		return nil
	}
	err := errors.New(C.GoStringN(error_msg.ptr, error_msg.len))
	C.free(unsafe.Pointer(error_msg.ptr))
	return err
}

// Context is a sandboxed js environment with its own set of built-in objects
// and functions.  Values and javascript operations within a context are visible
// only within that context unless the Go code explicitly moves values from one
// context to another.
type Context struct {
	id  int
	iso *Isolate
	ptr C.ContextPtr

	callbacks      map[int]callbackInfo
	nextCallbackId int

	values map[*Value]bool
}
type callbackInfo struct {
	Callback
	name string
}

func (ctx *Context) split(ret C.ValueErrorPair) (*Value, error) {
	return ctx.newValue(ret.Value), ctx.iso.convertErrorMsg(ret.error_msg)
}

// Eval runs the javascript code in the VM.  The filename parameter is
// informational only -- it is shown in javascript stack traces.
func (ctx *Context) Eval(jsCode, filename string) (*Value, error) {
	js_code_cstr := C.CString(jsCode)
	filename_cstr := C.CString(filename)
	ret := C.v8_Context_Run(ctx.ptr, js_code_cstr, filename_cstr)
	C.free(unsafe.Pointer(js_code_cstr))
	C.free(unsafe.Pointer(filename_cstr))
	return ctx.split(ret)
}

// Bind creates a V8 function value that calls a Go function when invoked.  This
// value is created but NOT visible in the Context until it is explicitly passed
// to the Context (either via a .Set() call or as a callback return value).
func (ctx *Context) Bind(name string, cb Callback) *Value {
	ctx.nextCallbackId++
	id := ctx.nextCallbackId
	ctx.callbacks[id] = callbackInfo{cb, name}
	cbIdStr := C.CString(fmt.Sprintf("%d:%d", ctx.id, id))
	defer C.free(unsafe.Pointer(cbIdStr))
	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))
	return ctx.newValue(C.v8_Context_RegisterCallback(ctx.ptr, nameStr, cbIdStr))
}

// Global returns the JS global object for this context, with properties like
// Object, Array, JSON, etc.
func (ctx *Context) Global() *Value {
	return ctx.newValue(C.v8_Context_Global(ctx.ptr))
}
func (ctx *Context) release() {
	for val := range ctx.values {
		val.release()
	}
	if ctx.ptr != nil {
		C.v8_Context_Release(ctx.ptr)
	}
	ctx.ptr = nil
	contextsMutex.Lock()
	delete(contexts, ctx.id)
	contextsMutex.Unlock()
}

// Terminate will interrupt any processing going on in the context.  This may
// be called from any goroutine.
func (ctx *Context) Terminate() { ctx.iso.Terminate() }
func (ctx *Context) newValue(ptr C.PersistentValuePtr) *Value {
	if ptr == nil {
		return nil
	}

	val := &Value{ctx, ptr}
	// Track allocated Persistent values so we can clean up.
	ctx.values[val] = true
	runtime.SetFinalizer(val, (*Value).release)
	return val
}

// ParseJson uses V8's JSON.parse to parse the string and return the parsed
// object.
func (ctx *Context) ParseJson(json string) (*Value, error) {
	var json_parse *Value
	if json, err := ctx.Global().Get("JSON"); err != nil {
		return nil, fmt.Errorf("Cannot get JSON: %v", err)
	} else if json_parse, err = json.Get("parse"); err != nil {
		return nil, fmt.Errorf("Cannot get JSON.parse: %v", err)
	}
	str, err := ctx.Create(json)
	if err != nil {
		return nil, err
	}
	return json_parse.Call(json_parse, str)
}

// Value represents a handle to a value within the javascript VM.  Values are
// associated with a particular Context, but may be passed freely between
// Contexts within an Isolate.
type Value struct {
	ctx *Context
	ptr C.PersistentValuePtr
}

// String returns the string representation of the value using the ToString()
// method.  For primitive types this is just the printable value.  For objects,
// this is "[object Object]".  Functions print the function definition.
func (v *Value) String() string {
	cstr := C.v8_Value_String(v.ctx.ptr, v.ptr)
	str := C.GoStringN(cstr.ptr, cstr.len)
	C.free(unsafe.Pointer(cstr.ptr))
	return str
}

// Get a field from the object.  If this value is not an object, this will fail.
func (v *Value) Get(name string) (*Value, error) {
	name_cstr := C.CString(name)
	ret := C.v8_Value_Get(v.ctx.ptr, v.ptr, name_cstr)
	C.free(unsafe.Pointer(name_cstr))
	return v.ctx.split(ret)
}

// Get the value at the specified index.  If this value is not an object or an
// array, this will fail.
func (v *Value) GetIndex(idx int) (*Value, error) {
	return v.ctx.split(C.v8_Value_GetIdx(v.ctx.ptr, v.ptr, C.int(idx)))
}

// Set a field on the object.  If this value is not an object, this
// will fail.
func (v *Value) Set(name string, value *Value) error {
	name_cstr := C.CString(name)
	errmsg := C.v8_Value_Set(v.ctx.ptr, v.ptr, name_cstr, value.ptr)
	C.free(unsafe.Pointer(name_cstr))
	return v.ctx.iso.convertErrorMsg(errmsg)
}

// SetIndex sets the object's value at the specified index.  If this value is
// not an object or an array, this will fail.
func (v *Value) SetIndex(idx int, value *Value) error {
	return v.ctx.iso.convertErrorMsg(
		C.v8_Value_SetIdx(v.ctx.ptr, v.ptr, C.int(idx), value.ptr))
}

// Call this value as a function.  If this value is not a function, this will
// fail.
func (v *Value) Call(this *Value, args ...*Value) (*Value, error) {
	// always allocate at least one so &argPtrs[0] works.
	argPtrs := make([]C.PersistentValuePtr, len(args)+1)
	for i := range args {
		argPtrs[i] = args[i].ptr
	}
	var thisPtr C.PersistentValuePtr
	if this != nil {
		thisPtr = this.ptr
	}
	result := C.v8_Value_Call(v.ctx.ptr, v.ptr, thisPtr, C.int(len(args)), &argPtrs[0])
	return v.ctx.split(result)
}

// New creates a new instance of an object using this value as its constructor.
// If this value is not a function, this will fail.
func (v *Value) New(args ...*Value) (*Value, error) {
	// always allocate at least one so &argPtrs[0] works.
	argPtrs := make([]C.PersistentValuePtr, len(args)+1)
	for i := range args {
		argPtrs[i] = args[i].ptr
	}
	result := C.v8_Value_New(v.ctx.ptr, v.ptr, C.int(len(args)), &argPtrs[0])
	return v.ctx.split(result)
}

func (v *Value) release() {
	if v.ctx != nil {
		delete(v.ctx.values, v)
	}
	if v.ptr != nil {
		C.v8_Value_Release(v.ctx.ptr, v.ptr)
	}
	v.ctx = nil
	v.ptr = nil
}

// MarshalJSON implements the json.Marshaler interface using the JSON.stringify
// function from the VM to serialize the value and fails if that cannot be
// found.
//
// Note that JSON.stringify will ignore function values.  For example, this JS
// object:
//   { foo: function() { return "x" }, bar: 3 }
// will serialize to this:
//   {"bar":3}
func (v *Value) MarshalJSON() ([]byte, error) {
	var json_stringify *Value
	if json, err := v.ctx.Global().Get("JSON"); err != nil {
		return nil, fmt.Errorf("Cannot get JSON object: %v", err)
	} else if json_stringify, err = json.Get("stringify"); err != nil {
		return nil, fmt.Errorf("Cannot get JSON.stringify: %v", err)
	}
	res, err := json_stringify.Call(json_stringify, v)
	if err != nil {
		return nil, fmt.Errorf("Failed to stringify val: %v", err)
	}
	return []byte(res.String()), nil
}

//
//
//

var contexts = map[int]*Context{}
var contextsMutex sync.RWMutex
var nextContextId int

//export go_callback_handler
func go_callback_handler(
	cbIdStr C.String,
	caller C.CallerInfo,
	argc C.int,
	argvptr C.PersistentValuePtr,
) (ret C.ValueErrorPair) {
	caller_loc := Loc{
		Funcname: C.GoStringN(caller.Funcname.ptr, caller.Funcname.len),
		Filename: C.GoStringN(caller.Filename.ptr, caller.Filename.len),
		Line:     int(caller.Line),
		Column:   int(caller.Column),
	}

	cbId := C.GoStringN(cbIdStr.ptr, cbIdStr.len)
	parts := strings.SplitN(cbId, ":", 2)
	ctxId, _ := strconv.Atoi(parts[0])
	callbackId, _ := strconv.Atoi(parts[1])

	contextsMutex.RLock()
	ctx := contexts[ctxId]
	contextsMutex.RUnlock()

	info := ctx.callbacks[int(callbackId)]
	if info.Callback == nil {
		// Everything is bad -- this should never happen.
		panic(fmt.Errorf("No such registered callback: %s", info.name))
	}

	// Convert array of args into a slice.  See:
	//   https://github.com/golang/go/wiki/cgo
	// and
	//   http://play.golang.org/p/XuC0xqtAIC
	argv := (*[1 << 30]C.PersistentValuePtr)(unsafe.Pointer(argvptr))[:argc:argc]

	args := make([]*Value, argc)
	for i := 0; i < int(argc); i++ {
		args[i] = ctx.newValue(argv[i])
	}

	// Catch panics -- if they are uncaught, they skip past the C stack and
	// continue straight through to the go call, wreaking havoc with the C
	// state.
	defer func() {
		if v := recover(); v != nil {
			errmsg := fmt.Sprintf("Panic during callback %q: %v", info.name, v)
			ret.error_msg = C.Error{ptr: C.CString(errmsg), len: C.int(len(errmsg))}
		}
	}()

	res, err := info.Callback(CallbackArgs{caller_loc, args, ctx})

	if err != nil {
		errmsg := err.Error()
		e := C.Error{ptr: C.CString(errmsg), len: C.int(len(errmsg))}
		return C.ValueErrorPair{nil, e}
	}

	if res == nil {
		return C.ValueErrorPair{}
	} else if res.ctx.iso.ptr != ctx.iso.ptr {
		errmsg := fmt.Sprintf("Callback %s returned a value from another isolate.", info.name)
		e := C.Error{ptr: C.CString(errmsg), len: C.int(len(errmsg))}
		return C.ValueErrorPair{nil, e}
	}

	return C.ValueErrorPair{Value: res.ptr}
}
