package v8

import (
	"fmt"
)

type Kind uint8

// Value kinds
const (
	KindUndefined Kind = iota
	KindNull
	KindTrue
	KindFalse
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
)

func (k Kind) String() string {
	switch k {
	case KindUndefined:
		return "Undefined"
	case KindNull:
		return "Null"
	case KindTrue:
		return "True"
	case KindFalse:
		return "False"
	case KindName:
		return "Name"
	case KindString:
		return "String"
	case KindSymbol:
		return "Symbol"
	case KindFunction:
		return "Function"
	case KindArray:
		return "Array"
	case KindObject:
		return "Object"
	case KindBoolean:
		return "Boolean"
	case KindNumber:
		return "Number"
	case KindExternal:
		return "External"
	case KindInt32:
		return "Int32"
	case KindUint32:
		return "Uint32"
	case KindDate:
		return "Date"
	case KindArgumentsObject:
		return "ArgumentsObject"
	case KindBooleanObject:
		return "BooleanObject"
	case KindNumberObject:
		return "NumberObject"
	case KindStringObject:
		return "StringObject"
	case KindSymbolObject:
		return "SymbolObject"
	case KindNativeError:
		return "NativeError"
	case KindRegExp:
		return "RegExp"
	case KindAsyncFunction:
		return "AsyncFunction"
	case KindGeneratorFunction:
		return "GeneratorFunction"
	case KindGeneratorObject:
		return "GeneratorObject"
	case KindPromise:
		return "Promise"
	case KindMap:
		return "Map"
	case KindSet:
		return "Set"
	case KindMapIterator:
		return "MapIterator"
	case KindSetIterator:
		return "SetIterator"
	case KindWeakMap:
		return "WeakMap"
	case KindWeakSet:
		return "WeakSet"
	case KindArrayBuffer:
		return "ArrayBuffer"
	case KindArrayBufferView:
		return "ArrayBufferView"
	case KindTypedArray:
		return "TypedArray"
	case KindUint8Array:
		return "Uint8Array"
	case KindUint8ClampedArray:
		return "Uint8ClampedArray"
	case KindInt8Array:
		return "Int8Array"
	case KindUint16Array:
		return "Uint16Array"
	case KindInt16Array:
		return "Int16Array"
	case KindUint32Array:
		return "Uint32Array"
	case KindInt32Array:
		return "Int32Array"
	case KindFloat32Array:
		return "Float32Array"
	case KindFloat64Array:
		return "Float64Array"
	case KindDataView:
		return "DataView"
	case KindSharedArrayBuffer:
		return "SharedArrayBuffer"
	case KindProxy:
		return "Proxy"
	case KindWebAssemblyCompiledModule:
		return "WebAssemblyCompiledModule"
	}
	return fmt.Sprintf("InvalidKind:%d", int(k))
}

// Value kind unions, most values have multiple kinds
var (
	unionKindString          = []Kind{KindName, KindString}
	unionKindSymbol          = []Kind{KindName, KindSymbol}
	unionKindFunction        = []Kind{KindObject, KindFunction}
	unionKindArray           = []Kind{KindObject, KindArray}
	unionKindTrue            = []Kind{KindBoolean, KindTrue}
	unionKindFalse           = []Kind{KindBoolean, KindFalse}
	unionKindDate            = []Kind{KindObject, KindDate}
	unionKindArgumentsObject = []Kind{KindObject, KindArgumentsObject}

	unionKindBooleanObject     = []Kind{KindObject, KindBooleanObject}
	unionKindNumberObject      = []Kind{KindObject, KindNumberObject}
	unionKindStringObject      = []Kind{KindObject, KindStringObject}
	unionKindSymbolObject      = []Kind{KindObject, KindSymbolObject}
	unionKindRegExp            = []Kind{KindObject, KindRegExp}
	unionKindPromise           = []Kind{KindObject, KindPromise}
	unionKindMap               = []Kind{KindObject, KindMap}
	unionKindSet               = []Kind{KindObject, KindSet}
	unionKindArrayBuffer       = []Kind{KindObject, KindArrayBuffer}
	unionKindUint8Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint8Array}
	unionKindUint8ClampedArray = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint8ClampedArray}
	unionKindInt8Array         = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt8Array}
	unionKindUint16Array       = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint16Array}
	unionKindInt16Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt16Array}
	unionKindUint32Array       = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint32Array}
	unionKindInt32Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt32Array}
	unionKindFloat32Array      = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindFloat32Array}
	unionKindFloat64Array      = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindFloat64Array}
	unionKindDataView          = []Kind{KindObject, KindArrayBufferView, KindDataView}
	unionKindSharedArrayBuffer = []Kind{KindObject, KindSharedArrayBuffer}
	unionKindProxy             = []Kind{KindObject, KindProxy}
	unionKindWeakMap           = []Kind{KindObject, KindWeakMap}
	unionKindWeakSet           = []Kind{KindObject, KindWeakSet}
	unionKindAsyncFunction     = []Kind{KindObject, KindFunction, KindAsyncFunction}
	unionKindGeneratorFunction = []Kind{KindObject, KindFunction, KindGeneratorFunction}
	unionKindGeneratorObject   = []Kind{KindObject, KindGeneratorObject}
	unionKindMapIterator       = []Kind{KindObject, KindMapIterator}
	unionKindSetIterator       = []Kind{KindObject, KindSetIterator}
	unionKindNativeError       = []Kind{KindObject, KindNativeError}

	unionKindWebAssemblyCompiledModule = []Kind{KindObject, KindWebAssemblyCompiledModule}
)
