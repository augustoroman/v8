package v8

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
