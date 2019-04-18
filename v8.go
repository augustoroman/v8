package v8

// Reference materials:
//   https://developers.google.com/v8/embed#accessors
//   https://developers.google.com/v8/embed#exceptions
//   https://docs.google.com/document/d/1g8JFi8T_oAE_7uAri7Njtig7fKaPDfotU6huOa1alds/edit
// TODO:
//   Value.Export(v) --> inverse of Context.Create()
//   Proxy objects

// BUG(aroman) Unhandled promise rejections are silently dropped
// (see https://github.com/augustoroman/v8/issues/21)

// #include <stdlib.h>
// #include <string.h>
// #include "v8_c_bridge.h"
// #cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -fno-rtti -fpic -std=c++11
// #cgo LDFLAGS: -pthread -L${SRCDIR}/libv8 -lv8_base -lv8_init -lv8_initializers -lv8_libbase -lv8_libplatform -lv8_libsampler -lv8_nosnapshot
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
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

// PromiseState defines the state of a promise: either pending, resolved, or
// rejected. Promises that are pending have no result value yet. A promise that
// is resolved has a result value, and a promise that is rejected has a result
// value that is usually the error.
type PromiseState uint8

const (
	PromiseStatePending PromiseState = iota
	PromiseStateResolved
	PromiseStateRejected
	kNumPromiseStates
)

var promiseStateStrings = [kNumPromiseStates]string{"Pending", "Resolved", "Rejected"}

func (s PromiseState) String() string {
	if s < 0 || s >= kNumPromiseStates {
		return fmt.Sprintf("InvalidPromiseState:%d", int(s))
	}
	return promiseStateStrings[s]
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
	runtime.SetFinalizer(s, nil)
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
type Isolate struct {
	ptr C.IsolatePtr
	s   *Snapshot // make sure not to be advanced GC
}

// NewIsolate creates a new V8 Isolate.
func NewIsolate() *Isolate {
	v8_init_once.Do(func() { C.v8_init() })
	iso := &Isolate{ptr: C.v8_Isolate_New(C.StartupData{ptr: nil, len: 0})}
	runtime.SetFinalizer(iso, (*Isolate).release)
	return iso
}

// NewIsolateWithSnapshot creates a new V8 Isolate using the supplied Snapshot
// to initialize all Contexts created from this Isolate.
func NewIsolateWithSnapshot(s *Snapshot) *Isolate {
	v8_init_once.Do(func() { C.v8_init() })
	iso := &Isolate{ptr: C.v8_Isolate_New(s.data), s: s}
	runtime.SetFinalizer(iso, (*Isolate).release)
	return iso
}

// NewContext creates a new, clean V8 Context within this Isolate.
func (i *Isolate) NewContext() *Context {
	ctx := &Context{
		iso:       i,
		ptr:       C.v8_Isolate_NewContext(i.ptr),
		callbacks: map[int]callbackInfo{},
	}

	contextsMutex.Lock()
	nextContextId++
	ctx.id = nextContextId
	contextsMutex.Unlock()

	runtime.SetFinalizer(ctx, (*Context).release)

	return ctx
}

// Terminate will interrupt all operation in this Isolate, interrupting any
// Contexts that are executing.  This may be called from any goroutine at any
// time.
func (i *Isolate) Terminate() { C.v8_Isolate_Terminate(i.ptr) }
func (i *Isolate) release() {
	C.v8_Isolate_Release(i.ptr)
	i.ptr = nil
	runtime.SetFinalizer(i, nil)
}

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
}
type callbackInfo struct {
	Callback
	name string
}

func (ctx *Context) split(ret C.ValueTuple) (*Value, error) {
	return ctx.newValue(ret.Value, ret.Kinds), ctx.iso.convertErrorMsg(ret.error_msg)
}

// Eval runs the javascript code in the VM.  The filename parameter is
// informational only -- it is shown in javascript stack traces.
func (ctx *Context) Eval(jsCode, filename string) (*Value, error) {
	js_code_cstr := C.CString(jsCode)
	filename_cstr := C.CString(filename)
	addRef(ctx)
	ret := C.v8_Context_Run(ctx.ptr, js_code_cstr, filename_cstr)
	decRef(ctx)
	C.free(unsafe.Pointer(js_code_cstr))
	C.free(unsafe.Pointer(filename_cstr))
	return ctx.split(ret)
}

// Bind creates a V8 function value that calls a Go function when invoked. This
// value is created but NOT visible in the Context until it is explicitly passed
// to the Context (either via a .Set() call or as a callback return value).
//
// The name that is provided is the name of the defined javascript function, and
// generally doesn't affect anything. That is, for a call such as:
//
//     val, _ = ctx.Bind("my_func_name", callback)
//
// then val is a function object in javascript, so calling val.String() (or
// calling .toString() on the object within the JS VM) would result in:
//
//     function my_func_name() { [native code] }
//
// NOTE: Once registered, a callback function will be stored in the Context
// until it is GC'd, so each Bind for a given context will take up a little
// more memory each time. Normally this isn't a problem, but many many Bind's
// on a Context can gradually consume memory.
func (ctx *Context) Bind(name string, cb Callback) *Value {
	ctx.nextCallbackId++
	id := ctx.nextCallbackId
	ctx.callbacks[id] = callbackInfo{cb, name}
	cbIdStr := C.CString(fmt.Sprintf("%d:%d", ctx.id, id))
	defer C.free(unsafe.Pointer(cbIdStr))
	nameStr := C.CString(name)
	defer C.free(unsafe.Pointer(nameStr))
	return ctx.newValue(
		C.v8_Context_RegisterCallback(ctx.ptr, nameStr, cbIdStr),
		unionKindFunction,
	)
}

// Global returns the JS global object for this context, with properties like
// Object, Array, JSON, etc.
func (ctx *Context) Global() *Value {
	return ctx.newValue(C.v8_Context_Global(ctx.ptr), C.KindMask(KindObject))
}
func (ctx *Context) release() {
	if ctx.ptr != nil {
		C.v8_Context_Release(ctx.ptr)
	}
	ctx.ptr = nil

	contextsMutex.Lock()
	delete(contexts, ctx.id)
	contextsMutex.Unlock()

	runtime.SetFinalizer(ctx, nil)
	ctx.iso = nil // Allow the isolate to be GC'd if we're the last ptr to it.
}

// Terminate will interrupt any processing going on in the context.  This may
// be called from any goroutine.
func (ctx *Context) Terminate() { ctx.iso.Terminate() }
func (ctx *Context) newValue(ptr C.PersistentValuePtr, kinds C.KindMask) *Value {
	if ptr == nil {
		return nil
	}

	val := &Value{ctx, ptr, kindMask(kinds)}
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
	ctx      *Context
	ptr      C.PersistentValuePtr
	kindMask kindMask
}

// Bytes returns a byte slice extracted from this value when the value
// is of type ArrayBuffer. The returned byte slice is copied from the underlying
// buffer, so modifying it will not be reflected in the VM.
// Values of other types return nil.
func (v *Value) Bytes() []byte {
	mem := C.v8_Value_Bytes(v.ctx.ptr, v.ptr)
	if mem.ptr == nil {
		return nil
	}
	ret := make([]byte, mem.len)
	copy(ret, ((*[1 << 30]byte)(unsafe.Pointer(mem.ptr)))[:mem.len:mem.len])
	// NOTE: We don't free the memory here: It's owned by V8.
	return ret
}

// Float64 returns this Value as a float64. If this value is not a number,
// then NaN will be returned.
func (v *Value) Float64() float64 {
	return float64(C.v8_Value_Float64(v.ctx.ptr, v.ptr))
}

// Int64 returns this Value as an int64. If this value is not a number,
// then 0 will be returned.
func (v *Value) Int64() int64 {
	return int64(C.v8_Value_Int64(v.ctx.ptr, v.ptr))
}

// Bool returns this Value as a boolean. If the underlying value is not a
// boolean, it will be coerced to a boolean using Javascript's coercion rules.
func (v *Value) Bool() bool {
	return C.v8_Value_Bool(v.ctx.ptr, v.ptr) == 1
}

// Date returns this Value as a time.Time. If the underlying value is not a
// KindDate, this will return an error.
func (v *Value) Date() (time.Time, error) {
	if !v.IsKind(KindDate) {
		return time.Time{}, errors.New("Not a date")
	}
	msec := v.Int64()
	sec := msec / 1000
	nsec := (msec % 1000) * 1e6
	return time.Unix(sec, nsec), nil
}

// PromiseInfo will return information about the promise if this value's
// underlying kind is KindPromise, otherwise it will return an error. If there
// is no error, then the returned value will depend on the promise state:
//   pending: nil
//   fulfilled: the value of the promise
//   rejected: the rejected result, usually a JS error
func (v *Value) PromiseInfo() (PromiseState, *Value, error) {
	if !v.IsKind(KindPromise) {
		return 0, nil, errors.New("Not a promise")
	}
	var state C.int
	val, err := v.ctx.split(C.v8_Value_PromiseInfo(v.ctx.ptr, v.ptr, &state))
	return PromiseState(state), val, err
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
	addRef(v.ctx)
	result := C.v8_Value_Call(v.ctx.ptr, v.ptr, thisPtr, C.int(len(args)), &argPtrs[0])
	decRef(v.ctx)
	return v.ctx.split(result)
}

// IsKind will test whether the underlying value is the specified JS kind.
// The kind of a value is set when the value is created and will not change.
func (v *Value) IsKind(k Kind) bool {
	return v.kindMask.Is(k)
}

// New creates a new instance of an object using this value as its constructor.
// If this value is not a function, this will fail.
func (v *Value) New(args ...*Value) (*Value, error) {
	// always allocate at least one so &argPtrs[0] works.
	argPtrs := make([]C.PersistentValuePtr, len(args)+1)
	for i := range args {
		argPtrs[i] = args[i].ptr
	}
	addRef(v.ctx)
	result := C.v8_Value_New(v.ctx.ptr, v.ptr, C.int(len(args)), &argPtrs[0])
	decRef(v.ctx)
	return v.ctx.split(result)
}

func (v *Value) release() {
	if v.ptr != nil {
		C.v8_Value_Release(v.ctx.ptr, v.ptr)
	}
	v.ctx = nil
	v.ptr = nil
	runtime.SetFinalizer(v, nil)
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
// callback magic
//

// Because of the rules of Go <--> C pointer interchange
// (https://golang.org/cmd/cgo/#hdr-Passing_pointers), we can't pass a *Context
// pointer into the C code. That means that when V8 wants to execute a callback
// back into the Go code, we have to find some other way of determining which
// context the callback was associated with.
//
// One way (as described at https://github.com/golang/go/wiki/cgo) is to create
// a registry that keeps the pointers all in Go and uses an arbitrary numeric
// handle to pass to C instead.
//
// One tricky side-affect is that this holds a pointer to our Context. Well,
// that's obvious, right? But that means our Context can't be GC'd. Oops.
//
// To work around this, we'll dynamically create the registered entry each time
// we call into V8 and remove when we're done. Specifically, we'll use a ref
// count just in case somebody gets cute and calls back into V8 from a callback.
//
var contexts = map[int]*refCount{}
var contextsMutex sync.RWMutex
var nextContextId int

type refCount struct {
	ptr   *Context
	count int
}

func addRef(ctx *Context) {
	contextsMutex.Lock()
	ref := contexts[ctx.id]
	if ref == nil {
		ref = &refCount{ctx, 0}
		contexts[ctx.id] = ref
	}
	ref.count++
	contextsMutex.Unlock()
}
func decRef(ctx *Context) {
	contextsMutex.Lock()
	ref := contexts[ctx.id]
	if ref == nil || ref.count <= 1 {
		delete(contexts, ctx.id)
	} else {
		ref.count--
	}
	contextsMutex.Unlock()
}

//export go_callback_handler
func go_callback_handler(
	cbIdStr C.String,
	caller C.CallerInfo,
	argc C.int,
	argvptr *C.ValueTuple,
) (ret C.ValueTuple) {
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
	ref := contexts[ctxId]
	if ref == nil {
		panic(fmt.Errorf(
			"Missing context pointer during callback for context #%d", ctxId))
	}
	ctx := ref.ptr
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
	argv := (*[1 << 30]C.ValueTuple)(unsafe.Pointer(argvptr))[:argc:argc]

	args := make([]*Value, argc)
	for i := 0; i < int(argc); i++ {
		args[i] = ctx.newValue(argv[i].Value, argv[i].Kinds)
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
		return C.ValueTuple{nil, 0, e}
	}

	if res == nil {
		return C.ValueTuple{}
	} else if res.ctx.iso.ptr != ctx.iso.ptr {
		errmsg := fmt.Sprintf("Callback %s returned a value from another isolate.", info.name)
		e := C.Error{ptr: C.CString(errmsg), len: C.int(len(errmsg))}
		return C.ValueTuple{nil, 0, e}
	}

	return C.ValueTuple{Value: res.ptr}
}

// HeapStatistics represent v8::HeapStatistics which are statistics
// about the heap memory usage.
type HeapStatistics struct {
	TotalHeapSize           uint64
	TotalHeapSizeExecutable uint64
	TotalPhysicalSize       uint64
	TotalAvailableSize      uint64
	UsedHeapSize            uint64
	HeapSizeLimit           uint64
	MallocedMemory          uint64
	PeakMallocedMemory      uint64
	DoesZapGarbage          bool
}

// GetHeapStatistics gets statistics about the heap memory usage.
func (i *Isolate) GetHeapStatistics() HeapStatistics {
	hs := C.v8_Isolate_GetHeapStatistics(i.ptr)
	return HeapStatistics{
		TotalHeapSize:           uint64(hs.total_heap_size),
		TotalHeapSizeExecutable: uint64(hs.total_heap_size_executable),
		TotalPhysicalSize:       uint64(hs.total_physical_size),
		TotalAvailableSize:      uint64(hs.total_available_size),
		UsedHeapSize:            uint64(hs.used_heap_size),
		HeapSizeLimit:           uint64(hs.heap_size_limit),
		MallocedMemory:          uint64(hs.malloced_memory),
		PeakMallocedMemory:      uint64(hs.peak_malloced_memory),
		DoesZapGarbage:          hs.does_zap_garbage == 1,
	}
}

// SendLowMemoryNotification sends an optional notification that the
// system is running low on memory. V8 uses these notifications to
// attempt to free memory.
func (i *Isolate) SendLowMemoryNotification() {
	C.v8_Isolate_LowMemoryNotification(i.ptr)
}
