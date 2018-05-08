package v8

import (
	"fmt"
	"strings"
)

// Kind is an underlying V8 representation of a *Value. Javascript values may
// have multiple underyling kinds. For example, a function will be both
// KindObject and KindFunction.
type Kind uint8

const (
	KindUndefined Kind = iota
	KindNull
	KindName
	KindString
	KindSymbol
	KindFunction
	KindArray
	KindObject
	KindBoolean
	KindNumber
	KindExternal
	KindInt32
	KindUint32
	KindDate
	KindArgumentsObject
	KindBooleanObject
	KindNumberObject
	KindStringObject
	KindSymbolObject
	KindNativeError
	KindRegExp
	KindAsyncFunction
	KindGeneratorFunction
	KindGeneratorObject
	KindPromise
	KindMap
	KindSet
	KindMapIterator
	KindSetIterator
	KindWeakMap
	KindWeakSet
	KindArrayBuffer
	KindArrayBufferView
	KindTypedArray
	KindUint8Array
	KindUint8ClampedArray
	KindInt8Array
	KindUint16Array
	KindInt16Array
	KindUint32Array
	KindInt32Array
	KindFloat32Array
	KindFloat64Array
	KindDataView
	KindSharedArrayBuffer
	KindProxy
	KindWebAssemblyCompiledModule

	kNumKinds
)

var kindStrings = [kNumKinds]string{
	"Undefined",
	"Null",
	"Name",
	"String",
	"Symbol",
	"Function",
	"Array",
	"Object",
	"Boolean",
	"Number",
	"External",
	"Int32",
	"Uint32",
	"Date",
	"ArgumentsObject",
	"BooleanObject",
	"NumberObject",
	"StringObject",
	"SymbolObject",
	"NativeError",
	"RegExp",
	"AsyncFunction",
	"GeneratorFunction",
	"GeneratorObject",
	"Promise",
	"Map",
	"Set",
	"MapIterator",
	"SetIterator",
	"WeakMap",
	"WeakSet",
	"ArrayBuffer",
	"ArrayBufferView",
	"TypedArray",
	"Uint8Array",
	"Uint8ClampedArray",
	"Int8Array",
	"Uint16Array",
	"Int16Array",
	"Uint32Array",
	"Int32Array",
	"Float32Array",
	"Float64Array",
	"DataView",
	"SharedArrayBuffer",
	"Proxy",
	"WebAssemblyCompiledModule",
}

func (k Kind) String() string {
	if k >= kNumKinds || k < 0 {
		return fmt.Sprintf("NoSuchKind:%d", int(k))
	}
	return kindStrings[int(k)]
}

func (k Kind) mask() kindMask { return kindMask(1 << k) }

type kindMask uint64

// if kNumKinds > 64, then this will fail at compile time.
const compileCheckThatNumKindsBitsFitInKindType = kindMask(1 << kNumKinds)

func (mask kindMask) Is(k Kind) bool {
	return (mask & k.mask()) != 0
}

func (mask kindMask) String() string {
	var res []string
	for k := Kind(0); k < kNumKinds; k++ {
		if mask.Is(k) {
			res = append(res, k.String())
		}
	}
	return strings.Join(res, ",")
}

func mask(kinds ...Kind) kindMask {
	var res kindMask
	for _, k := range kinds {
		res = res | k.mask()
	}
	return res
}

// Value kind unions, most values have multiple kinds
const (
	unionKindString          = (1 << KindName) | (1 << KindString)
	unionKindSymbol          = (1 << KindName) | (1 << KindSymbol)
	unionKindFunction        = (1 << KindObject) | (1 << KindFunction)
	unionKindArray           = (1 << KindObject) | (1 << KindArray)
	unionKindDate            = (1 << KindObject) | (1 << KindDate)
	unionKindArgumentsObject = (1 << KindObject) | (1 << KindArgumentsObject)

	unionKindBooleanObject     = (1 << KindObject) | (1 << KindBooleanObject)
	unionKindNumberObject      = (1 << KindObject) | (1 << KindNumberObject)
	unionKindStringObject      = (1 << KindObject) | (1 << KindStringObject)
	unionKindSymbolObject      = (1 << KindObject) | (1 << KindSymbolObject)
	unionKindRegExp            = (1 << KindObject) | (1 << KindRegExp)
	unionKindPromise           = (1 << KindObject) | (1 << KindPromise)
	unionKindMap               = (1 << KindObject) | (1 << KindMap)
	unionKindSet               = (1 << KindObject) | (1 << KindSet)
	unionKindArrayBuffer       = (1 << KindObject) | (1 << KindArrayBuffer)
	unionKindUint8Array        = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindUint8Array)
	unionKindUint8ClampedArray = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindUint8ClampedArray)
	unionKindInt8Array         = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindInt8Array)
	unionKindUint16Array       = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindUint16Array)
	unionKindInt16Array        = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindInt16Array)
	unionKindUint32Array       = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindUint32Array)
	unionKindInt32Array        = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindInt32Array)
	unionKindFloat32Array      = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindFloat32Array)
	unionKindFloat64Array      = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindTypedArray) | (1 << KindFloat64Array)
	unionKindDataView          = (1 << KindObject) | (1 << KindArrayBufferView) | (1 << KindDataView)
	unionKindSharedArrayBuffer = (1 << KindObject) | (1 << KindSharedArrayBuffer)
	unionKindProxy             = (1 << KindObject) | (1 << KindProxy)
	unionKindWeakMap           = (1 << KindObject) | (1 << KindWeakMap)
	unionKindWeakSet           = (1 << KindObject) | (1 << KindWeakSet)
	unionKindAsyncFunction     = (1 << KindObject) | (1 << KindFunction) | (1 << KindAsyncFunction)
	unionKindGeneratorFunction = (1 << KindObject) | (1 << KindFunction) | (1 << KindGeneratorFunction)
	unionKindGeneratorObject   = (1 << KindObject) | (1 << KindGeneratorObject)
	unionKindMapIterator       = (1 << KindObject) | (1 << KindMapIterator)
	unionKindSetIterator       = (1 << KindObject) | (1 << KindSetIterator)
	unionKindNativeError       = (1 << KindObject) | (1 << KindNativeError)

	unionKindWebAssemblyCompiledModule = (1 << KindObject) | (1 << KindWebAssemblyCompiledModule)
)
