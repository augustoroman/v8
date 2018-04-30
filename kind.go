package v8

type Kind int32

// Value kinds
const (
	KindUndefined Kind = iota
	KindNull
	KindNullOrUndefined
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

// Value kind unions, most values have multiple kinds
var (
	UnionKindUndefined       = []Kind{KindUndefined, KindNullOrUndefined}
	UnionKindNull            = []Kind{KindNull, KindNullOrUndefined}
	UnionKindString          = []Kind{KindName, KindString}
	UnionKindSymbol          = []Kind{KindName, KindSymbol}
	UnionKindFunction        = []Kind{KindObject, KindFunction}
	UnionKindArray           = []Kind{KindObject, KindArray}
	UnionKindTrue            = []Kind{KindBoolean, KindTrue}
	UnionKindFalse           = []Kind{KindBoolean, KindFalse}
	UnionKindDate            = []Kind{KindObject, KindDate}
	UnionKindArgumentsObject = []Kind{KindObject, KindArgumentsObject}

	UnionKindBooleanObject     = []Kind{KindObject, KindBooleanObject}
	UnionKindNumberObject      = []Kind{KindObject, KindNumberObject}
	UnionKindStringObject      = []Kind{KindObject, KindStringObject}
	UnionKindSymbolObject      = []Kind{KindObject, KindSymbolObject}
	UnionKindRegExp            = []Kind{KindObject, KindRegExp}
	UnionKindPromise           = []Kind{KindObject, KindPromise}
	UnionKindMap               = []Kind{KindObject, KindMap}
	UnionKindSet               = []Kind{KindObject, KindSet}
	UnionKindArrayBuffer       = []Kind{KindObject, KindArrayBuffer}
	UnionKindUint8Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint8Array}
	UnionKindUint8ClampedArray = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint8ClampedArray}
	UnionKindInt8Array         = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt8Array}
	UnionKindUint16Array       = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint16Array}
	UnionKindInt16Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt16Array}
	UnionKindUint32Array       = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindUint32Array}
	UnionKindInt32Array        = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindInt32Array}
	UnionKindFloat32Array      = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindFloat32Array}
	UnionKindFloat64Array      = []Kind{KindObject, KindArrayBufferView, KindTypedArray, KindFloat64Array}
	UnionKindDataView          = []Kind{KindObject, KindArrayBufferView, KindDataView}
	UnionKindSharedArrayBuffer = []Kind{KindObject, KindSharedArrayBuffer}
	UnionKindProxy             = []Kind{KindObject, KindProxy}
	UnionKindWeakMap           = []Kind{KindObject, KindWeakMap}
	UnionKindWeakSet           = []Kind{KindObject, KindWeakSet}
	UnionKindAsyncFunction     = []Kind{KindObject, KindFunction, KindAsyncFunction}
	UnionKindGeneratorFunction = []Kind{KindObject, KindFunction, KindGeneratorFunction}
	UnionKindGeneratorObject   = []Kind{KindObject, KindGeneratorObject}
	UnionKindMapIterator       = []Kind{KindObject, KindMapIterator}
	UnionKindSetIterator       = []Kind{KindObject, KindSetIterator}
	UnionKindNativeError       = []Kind{KindObject, KindNativeError}

	UnionKindWebAssemblyCompiledModule = []Kind{KindObject, KindWebAssemblyCompiledModule}
)
