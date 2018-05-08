package v8

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
	"unsafe"
)

// #include <stdlib.h>
// #include <string.h>
// #include "v8_c_bridge.h"
// #cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -fno-rtti -std=c++11
// #cgo LDFLAGS: -pthread -L${SRCDIR}/libv8 -lv8_base -lv8_init -lv8_initializers -lv8_libbase -lv8_libplatform -lv8_libsampler -lv8_nosnapshot
import "C"

var float64Type = reflect.TypeOf(float64(0))
var callbackType = reflect.TypeOf(Callback(nil))
var stringType = reflect.TypeOf(string(""))
var valuePtrType = reflect.TypeOf((*Value)(nil))
var timeType = reflect.TypeOf(time.Time{})

// Create maps Go values into corresponding JavaScript values. This value is
// created but NOT visible in the Context until it is explicitly passed to the
// Context (either via a .Set() call or as a callback return value).
//
// Create can automatically map the following types of values:
//   * bool
//   * all integers and floats are mapped to JS numbers (float64)
//   * strings
//   * maps (keys must be strings, values must be convertible)
//   * time.Time values (converted to js Date object)
//   * structs (exported field values must be convertible)
//   * slices of convertible types
//   * pointers to any convertible field
//   * v8.Callback function (automatically bind'd)
//   * *v8.Value (returned as-is)
//
// Any nil pointers are converted to undefined in JS.
//
// Values for elements in maps, structs, and slices may be any of the above
// types.
//
// When structs are being converted, any fields with json struct tags will
// respect the json naming entry.  For example:
//     var x = struct {
//        Ignored     string `json:"-"`
//        Renamed     string `json:"foo"`
//        DefaultName string `json:",omitempty"`
//        Bar         string
//     }{"a", "b", "c", "d"}
// will be converted as:
//    {
//        foo: "a",
//        DefaultName: "b",
//        Bar: "c",
//    }
// Also, embedded structs (or pointers-to-structs) will get inlined.
//
// Byte slices tagged as 'v8:"arraybuffer"' will be converted into a javascript
// ArrayBuffer object for more efficient conversion. For example:
//    var y = struct {
//        Buf     []byte `v8:"arraybuffer"`
//    }{[]byte{1,2,3}}
// will be converted as
//    {
//       Buf: new Uint8Array([1,2,3]).buffer
//    }
func (ctx *Context) Create(val interface{}) (*Value, error) {
	return ctx.create(reflect.ValueOf(val))
}

func (ctx *Context) createVal(v C.ImmediateValue, kinds kindMask) *Value {
	return ctx.newValue(C.v8_Context_Create(ctx.ptr, v), C.KindMask(kinds))
}

func getJsName(fieldName, jsonTag string) string {
	jsonName := strings.TrimSpace(strings.Split(jsonTag, ",")[0])
	if jsonName == "-" {
		return "" // skip this field
	}
	if jsonName == "" {
		return fieldName // use the default name
	}
	return jsonName // explict name specified
}

func (ctx *Context) create(val reflect.Value) (*Value, error) {
	return ctx.createWithTags(val, []string{})
}

func (ctx *Context) createWithTags(val reflect.Value, tags []string) (*Value, error) {
	if !val.IsValid() {
		return ctx.createVal(C.ImmediateValue{Type: C.tUNDEFINED}, mask(KindUndefined)), nil
	}

	if val.Type() == valuePtrType {
		return val.Interface().(*Value), nil
	} else if val.Type() == timeType {
		msec := C.double(val.Interface().(time.Time).UnixNano()) / 1e6
		return ctx.createVal(C.ImmediateValue{Type: C.tDATE, Float64: msec}, unionKindDate), nil
	}

	switch val.Kind() {
	case reflect.Bool:
		bval := C.int(0)
		if val.Bool() {
			bval = 1
		}
		return ctx.createVal(C.ImmediateValue{Type: C.tBOOL, Bool: bval}, mask(KindBoolean)), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		num := C.double(val.Convert(float64Type).Float())
		return ctx.createVal(C.ImmediateValue{Type: C.tFLOAT64, Float64: num}, mask(KindNumber)), nil
	case reflect.String:
		gostr := val.String()
		str := C.ByteArray{ptr: C.CString(gostr), len: C.int(len(gostr))}
		defer C.free(unsafe.Pointer(str.ptr))
		return ctx.createVal(C.ImmediateValue{Type: C.tSTRING, Mem: str}, unionKindString), nil
	case reflect.UnsafePointer, reflect.Uintptr:
		return nil, fmt.Errorf("Uintptr not supported: %#v", val.Interface())
	case reflect.Complex64, reflect.Complex128:
		return nil, fmt.Errorf("Complex not supported: %#v", val.Interface())
	case reflect.Chan:
		return nil, fmt.Errorf("Chan not supported: %#v", val.Interface())
	case reflect.Func:
		if val.Type().ConvertibleTo(callbackType) {
			name := path.Base(runtime.FuncForPC(val.Pointer()).Name())
			return ctx.Bind(name, val.Convert(callbackType).Interface().(Callback)), nil
		}
		return nil, fmt.Errorf("Func not supported: %#v", val.Interface())
	case reflect.Interface, reflect.Ptr:
		return ctx.create(val.Elem())
	case reflect.Map:
		if val.Type().Key() != stringType {
			return nil, fmt.Errorf("Map keys must be strings, %s not allowed", val.Type().Key())
		}
		ob := ctx.createVal(C.ImmediateValue{Type: C.tOBJECT}, mask(KindObject))
		keys := val.MapKeys()
		sort.Sort(stringKeys(keys))
		for _, key := range keys {
			v, err := ctx.create(val.MapIndex(key))
			if err != nil {
				return nil, fmt.Errorf("map key %q: %v", key.String(), err)
			}
			if err := ob.Set(key.String(), v); err != nil {
				return nil, err
			}
			v.release()
		}
		return ob, nil
	case reflect.Struct:
		ob := ctx.createVal(C.ImmediateValue{Type: C.tOBJECT}, mask(KindObject))
		return ob, ctx.writeStructFields(ob, val)
	case reflect.Array, reflect.Slice:
		arrayBuffer := false
		for _, tag := range tags {
			if strings.TrimSpace(tag) == "arraybuffer" {
				arrayBuffer = true
			}
		}

		if arrayBuffer && val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
			// Special case for byte array -> arraybuffer
			bytes := val.Bytes()
			var ptr *C.char
			if bytes != nil && len(bytes) > 0 {
				ptr = (*C.char)(unsafe.Pointer(&val.Bytes()[0]))
			}
			ob := ctx.createVal(
				C.ImmediateValue{
					Type: C.tARRAYBUFFER,
					Mem:  C.ByteArray{ptr: ptr, len: C.int(val.Len())},
				},
				unionKindArrayBuffer,
			)
			return ob, nil
		} else {
			ob := ctx.createVal(
				C.ImmediateValue{
					Type: C.tARRAY,
					Mem:  C.ByteArray{ptr: nil, len: C.int(val.Len())},
				},
				unionKindArray,
			)
			for i := 0; i < val.Len(); i++ {
				v, err := ctx.create(val.Index(i))
				if err != nil {
					return nil, fmt.Errorf("index %d: %v", i, err)
				}
				if err := ob.SetIndex(i, v); err != nil {
					return nil, err
				}
				v.release()
			}
			return ob, nil
		}
	}
	panic("Unknown kind!")
}

func (ctx *Context) writeStructFields(ob *Value, val reflect.Value) error {
	t := val.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := getJsName(f.Name, f.Tag.Get("json"))
		if name == "" {
			continue // skip field with tag `json:"-"`
		}

		// Inline embedded fields.
		if f.Anonymous {
			sub := val.Field(i)
			for sub.Kind() == reflect.Ptr && !sub.IsNil() {
				sub = sub.Elem()
			}

			if sub.Kind() == reflect.Struct {
				err := ctx.writeStructFields(ob, sub)
				if err != nil {
					return fmt.Errorf("Writing embedded field %q: %v", f.Name, err)
				}
				continue
			}
		}

		if !unicode.IsUpper(rune(f.Name[0])) {
			continue // skip unexported fields
		}

		v8Tags := strings.Split(f.Tag.Get("v8"), ",")
		v, err := ctx.createWithTags(val.Field(i), v8Tags)
		if err != nil {
			return fmt.Errorf("field %q: %v", f.Name, err)
		}
		if err := ob.Set(name, v); err != nil {
			return err
		}
		v.release()
	}

	// Also export any methods of the struct that match the callback type.
	for i := 0; i < t.NumMethod(); i++ {
		name := t.Method(i).Name
		if !unicode.IsUpper(rune(name[0])) {
			continue // skip unexported values
		}

		m := val.Method(i)
		if m.Type().ConvertibleTo(callbackType) {
			v, err := ctx.create(m)
			if err != nil {
				return fmt.Errorf("method %q: %v", name, err)
			}
			if err := ob.Set(name, v); err != nil {
				return err
			}
			v.release()
		}
	}
	return nil
}

type stringKeys []reflect.Value

func (s stringKeys) Len() int           { return len(s) }
func (s stringKeys) Swap(a, b int)      { s[a], s[b] = s[b], s[a] }
func (s stringKeys) Less(a, b int) bool { return s[a].String() < s[b].String() }
