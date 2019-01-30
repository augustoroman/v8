#include "v8_c_bridge.h"

#include "libplatform/libplatform.h"
#include "v8.h"

#include <cstdlib>
#include <cstring>
#include <string>
#include <sstream>
#include <stdio.h>

#define ISOLATE_SCOPE(iso) \
  v8::Isolate* isolate = (iso);                                                               \
  v8::Locker locker(isolate);                            /* Lock to current thread.        */ \
  v8::Isolate::Scope isolate_scope(isolate);             /* Assign isolate to this thread. */


#define VALUE_SCOPE(ctxptr) \
  ISOLATE_SCOPE(static_cast<Context*>(ctxptr)->isolate)                                       \
  v8::HandleScope handle_scope(isolate);                 /* Create a scope for handles.    */ \
  v8::Local<v8::Context> ctx(static_cast<Context*>(ctxptr)->ptr.Get(isolate));                \
  v8::Context::Scope context_scope(ctx);                 /* Scope to this context.         */

extern "C" ValueTuple go_callback_handler(
    String id, CallerInfo info, int argc, ValueTuple* argv);

// We only need one, it's stateless.
auto allocator = v8::ArrayBuffer::Allocator::NewDefaultAllocator();

typedef struct {
  v8::Persistent<v8::Context> ptr;
  v8::Isolate* isolate;
} Context;

typedef v8::Persistent<v8::Value> Value;

String DupString(const v8::String::Utf8Value& src) {
  char* data = static_cast<char*>(malloc(src.length()));
  memcpy(data, *src, src.length());
  return (String){data, src.length()};
}
String DupString(const v8::Local<v8::Value>& val) {
  return DupString(v8::String::Utf8Value(val));
}
String DupString(const char* msg) {
  const char* data = strdup(msg);
  return (String){data, int(strlen(msg))};
}
String DupString(const std::string& src) {
  char* data = static_cast<char*>(malloc(src.length()));
  memcpy(data, src.data(), src.length());
  return (String){data, int(src.length())};
}

KindMask v8_Value_KindsFromLocal(v8::Local<v8::Value> value) {
  KindMask kinds = 0;

  if (value->IsUndefined())         kinds |= (1ULL << Kind::kUndefined        );
  if (value->IsNull())              kinds |= (1ULL << Kind::kNull             );
  if (value->IsName())              kinds |= (1ULL << Kind::kName             );
  if (value->IsString())            kinds |= (1ULL << Kind::kString           );
  if (value->IsSymbol())            kinds |= (1ULL << Kind::kSymbol           );
  if (value->IsObject())            kinds |= (1ULL << Kind::kObject           );
  if (value->IsArray())             kinds |= (1ULL << Kind::kArray            );
  if (value->IsBoolean())           kinds |= (1ULL << Kind::kBoolean          );
  if (value->IsNumber())            kinds |= (1ULL << Kind::kNumber           );
  if (value->IsExternal())          kinds |= (1ULL << Kind::kExternal         );
  if (value->IsInt32())             kinds |= (1ULL << Kind::kInt32            );
  if (value->IsUint32())            kinds |= (1ULL << Kind::kUint32           );
  if (value->IsDate())              kinds |= (1ULL << Kind::kDate             );
  if (value->IsArgumentsObject())   kinds |= (1ULL << Kind::kArgumentsObject  );
  if (value->IsBooleanObject())     kinds |= (1ULL << Kind::kBooleanObject    );
  if (value->IsNumberObject())      kinds |= (1ULL << Kind::kNumberObject     );
  if (value->IsStringObject())      kinds |= (1ULL << Kind::kStringObject     );
  if (value->IsSymbolObject())      kinds |= (1ULL << Kind::kSymbolObject     );
  if (value->IsNativeError())       kinds |= (1ULL << Kind::kNativeError      );
  if (value->IsRegExp())            kinds |= (1ULL << Kind::kRegExp           );
  if (value->IsFunction())          kinds |= (1ULL << Kind::kFunction         );
  if (value->IsAsyncFunction())     kinds |= (1ULL << Kind::kAsyncFunction    );
  if (value->IsGeneratorFunction()) kinds |= (1ULL << Kind::kGeneratorFunction);
  if (value->IsGeneratorObject())   kinds |= (1ULL << Kind::kGeneratorObject  );
  if (value->IsPromise())           kinds |= (1ULL << Kind::kPromise          );
  if (value->IsMap())               kinds |= (1ULL << Kind::kMap              );
  if (value->IsSet())               kinds |= (1ULL << Kind::kSet              );
  if (value->IsMapIterator())       kinds |= (1ULL << Kind::kMapIterator      );
  if (value->IsSetIterator())       kinds |= (1ULL << Kind::kSetIterator      );
  if (value->IsWeakMap())           kinds |= (1ULL << Kind::kWeakMap          );
  if (value->IsWeakSet())           kinds |= (1ULL << Kind::kWeakSet          );
  if (value->IsArrayBuffer())       kinds |= (1ULL << Kind::kArrayBuffer      );
  if (value->IsArrayBufferView())   kinds |= (1ULL << Kind::kArrayBufferView  );
  if (value->IsTypedArray())        kinds |= (1ULL << Kind::kTypedArray       );
  if (value->IsUint8Array())        kinds |= (1ULL << Kind::kUint8Array       );
  if (value->IsUint8ClampedArray()) kinds |= (1ULL << Kind::kUint8ClampedArray);
  if (value->IsInt8Array())         kinds |= (1ULL << Kind::kInt8Array        );
  if (value->IsUint16Array())       kinds |= (1ULL << Kind::kUint16Array      );
  if (value->IsInt16Array())        kinds |= (1ULL << Kind::kInt16Array       );
  if (value->IsUint32Array())       kinds |= (1ULL << Kind::kUint32Array      );
  if (value->IsInt32Array())        kinds |= (1ULL << Kind::kInt32Array       );
  if (value->IsFloat32Array())      kinds |= (1ULL << Kind::kFloat32Array     );
  if (value->IsFloat64Array())      kinds |= (1ULL << Kind::kFloat64Array     );
  if (value->IsDataView())          kinds |= (1ULL << Kind::kDataView         );
  if (value->IsSharedArrayBuffer()) kinds |= (1ULL << Kind::kSharedArrayBuffer);
  if (value->IsProxy())             kinds |= (1ULL << Kind::kProxy            );
  if (value->IsWebAssemblyCompiledModule())
    kinds |= (1ULL << Kind::kWebAssemblyCompiledModule);

  return kinds;
}

std::string str(v8::Local<v8::Value> value) {
  v8::String::Utf8Value s(value);
  if (s.length() == 0) {
    return "";
  }
  return *s;
}

std::string report_exception(v8::Isolate* isolate, v8::Local<v8::Context> ctx, v8::TryCatch& try_catch) {
  std::stringstream ss;
  ss << "Uncaught exception: ";

  std::string exceptionStr = str(try_catch.Exception());
  ss << exceptionStr; // TODO(aroman) JSON-ify objects?

  if (!try_catch.Message().IsEmpty()) {
    if (!try_catch.Message()->GetScriptResourceName()->IsUndefined()) {
      ss << std::endl
         << "at " << str(try_catch.Message()->GetScriptResourceName());

      v8::Maybe<int> line_no = try_catch.Message()->GetLineNumber(ctx);
      v8::Maybe<int> start = try_catch.Message()->GetStartColumn(ctx);
      v8::Maybe<int> end = try_catch.Message()->GetEndColumn(ctx);
      v8::MaybeLocal<v8::String> sourceLine = try_catch.Message()->GetSourceLine(ctx);

      if (line_no.IsJust()) {
        ss << ":" << line_no.ToChecked();
      }
      if (start.IsJust()) {
        ss << ":" << start.ToChecked();
      }
      if (!sourceLine.IsEmpty()) {
        ss << std::endl
           << "  " << str(sourceLine.ToLocalChecked());
      }
      if (start.IsJust() && end.IsJust()) {
        ss << std::endl
           << "  ";
        for (int i = 0; i < start.ToChecked(); i++) {
          ss << " ";
        }
        for (int i = start.ToChecked(); i < end.ToChecked(); i++) {
          ss << "^";
        }
      }
    }
  }

  if (!try_catch.StackTrace().IsEmpty()) {
    ss << std::endl << "Stack trace: " << str(try_catch.StackTrace());
  }

  return ss.str();
}


extern "C" {

Version version = {V8_MAJOR_VERSION, V8_MINOR_VERSION, V8_BUILD_NUMBER, V8_PATCH_LEVEL};

void v8_init() {
  v8::Platform *platform = v8::platform::CreateDefaultPlatform(
      0, // thread_pool_size
      v8::platform::IdleTaskSupport::kDisabled,
      v8::platform::InProcessStackDumping::kDisabled);
  v8::V8::InitializePlatform(platform);
  v8::V8::Initialize();
  return;
}

StartupData v8_CreateSnapshotDataBlob(const char* js) {
  v8::StartupData data = v8::V8::CreateSnapshotDataBlob(js);
  return StartupData{data.data, data.raw_size};
}

IsolatePtr v8_Isolate_New(StartupData startup_data) {
  v8::Isolate::CreateParams create_params;
  create_params.array_buffer_allocator = allocator;
  if (startup_data.len > 0 && startup_data.ptr != nullptr) {
    v8::StartupData* data = new v8::StartupData;
    data->data = startup_data.ptr;
    data->raw_size = startup_data.len;
    create_params.snapshot_blob = data;
  }
  return static_cast<IsolatePtr>(v8::Isolate::New(create_params));
}
ContextPtr v8_Isolate_NewContext(IsolatePtr isolate_ptr) {
  ISOLATE_SCOPE(static_cast<v8::Isolate*>(isolate_ptr));
  v8::HandleScope handle_scope(isolate);

  isolate->SetCaptureStackTraceForUncaughtExceptions(true);

  v8::Local<v8::ObjectTemplate> globals = v8::ObjectTemplate::New(isolate);

  Context* ctx = new Context;
  ctx->ptr.Reset(isolate, v8::Context::New(isolate, nullptr, globals));
  ctx->isolate = isolate;
  return static_cast<ContextPtr>(ctx);
}
void v8_Isolate_Terminate(IsolatePtr isolate_ptr) {
  v8::Isolate* isolate = static_cast<v8::Isolate*>(isolate_ptr);
  isolate->TerminateExecution();
}
void v8_Isolate_Release(IsolatePtr isolate_ptr) {
  if (isolate_ptr == nullptr) {
    return;
  }
  v8::Isolate* isolate = static_cast<v8::Isolate*>(isolate_ptr);
  isolate->Dispose();
}

ValueTuple v8_Context_Run(ContextPtr ctxptr, const char* code, const char* filename) {
  Context* ctx = static_cast<Context*>(ctxptr);
  v8::Isolate* isolate = ctx->isolate;
  v8::Locker locker(isolate);
  v8::Isolate::Scope isolate_scope(isolate);
  v8::HandleScope handle_scope(isolate);
  v8::Context::Scope context_scope(ctx->ptr.Get(isolate));
  v8::TryCatch try_catch(isolate);
  try_catch.SetVerbose(false);

  filename = filename ? filename : "(no file)";

  ValueTuple res = { nullptr, 0, nullptr };

  v8::Local<v8::Script> script = v8::Script::Compile(
      v8::String::NewFromUtf8(isolate, code),
      v8::String::NewFromUtf8(isolate, filename));

  if (script.IsEmpty()) {
    res.error_msg = DupString(report_exception(isolate, ctx->ptr.Get(isolate), try_catch));
    return res;
  }

  v8::Local<v8::Value> result = script->Run();

  if (result.IsEmpty()) {
    res.error_msg = DupString(report_exception(isolate, ctx->ptr.Get(isolate), try_catch));
  } else {
    res.Value = static_cast<PersistentValuePtr>(new Value(isolate, result));
    res.Kinds = v8_Value_KindsFromLocal(result);
  }

	return res;
}

void go_callback(const v8::FunctionCallbackInfo<v8::Value>& args);

PersistentValuePtr v8_Context_RegisterCallback(
    ContextPtr ctxptr,
    const char* name,
    const char* id
) {
  VALUE_SCOPE(ctxptr);

  v8::Local<v8::FunctionTemplate> cb =
    v8::FunctionTemplate::New(isolate, go_callback,
      v8::String::NewFromUtf8(isolate, id));
  cb->SetClassName(v8::String::NewFromUtf8(isolate, name));
  return new Value(isolate, cb->GetFunction());
}

void go_callback(const v8::FunctionCallbackInfo<v8::Value>& args) {
  v8::Isolate* iso = args.GetIsolate();
  v8::HandleScope scope(iso);

  std::string id = str(args.Data());

  std::string src_file, src_func;
  int line_number = 0, column = 0;
  v8::Local<v8::StackTrace> trace(v8::StackTrace::CurrentStackTrace(iso, 1));
  if (trace->GetFrameCount() == 1) {
    v8::Local<v8::StackFrame> frame(trace->GetFrame(0));
    src_file = str(frame->GetScriptName());
    src_func = str(frame->GetFunctionName());
    line_number = frame->GetLineNumber();
    column = frame->GetColumn();
  }

  int argc = args.Length();
  ValueTuple argv[argc];
  for (int i = 0; i < argc; i++) {
    argv[i] = (ValueTuple){new Value(iso, args[i]), v8_Value_KindsFromLocal(args[i])};
  }

  ValueTuple result =
      go_callback_handler(
        (String){id.data(), int(id.length())},
        (CallerInfo){
          (String){src_func.data(), int(src_func.length())},
          (String){src_file.data(), int(src_file.length())},
          line_number,
          column
        },
        argc, argv);

  if (result.error_msg.ptr != nullptr) {
    v8::Local<v8::Value> err = v8::Exception::Error(
      v8::String::NewFromUtf8(iso, result.error_msg.ptr, v8::NewStringType::kNormal, result.error_msg.len).ToLocalChecked());
    iso->ThrowException(err);
  } else if (result.Value == NULL) {
    args.GetReturnValue().Set(v8::Undefined(iso));
  } else {
    args.GetReturnValue().Set(*static_cast<Value*>(result.Value));
  }
}

PersistentValuePtr v8_Context_Global(ContextPtr ctxptr) {
  VALUE_SCOPE(ctxptr);
  return new Value(isolate, ctx->Global());
}

void v8_Context_Release(ContextPtr ctxptr) {
  if (ctxptr == nullptr) {
    return;
  }
  Context* ctx = static_cast<Context*>(ctxptr);
  ISOLATE_SCOPE(ctx->isolate);
  ctx->ptr.Reset();
}

PersistentValuePtr v8_Context_Create(ContextPtr ctxptr, ImmediateValue val) {
  VALUE_SCOPE(ctxptr);

  switch (val.Type) {
    case tARRAY:       return new Value(isolate, v8::Array::New(isolate, val.Mem.len)); break;
    case tARRAYBUFFER: {
        v8::Local<v8::ArrayBuffer> buf = v8::ArrayBuffer::New(isolate, val.Mem.len);
        memcpy(buf->GetContents().Data(), val.Mem.ptr, val.Mem.len);
        return new Value(isolate, buf);
        break;
    }
    case tBOOL:        return new Value(isolate, v8::Boolean::New(isolate, val.Bool == 1)); break;
    case tDATE:        return new Value(isolate, v8::Date::New(isolate, val.Float64)); break;
    case tFLOAT64:     return new Value(isolate, v8::Number::New(isolate, val.Float64)); break;
    // For now, this is converted to a double on entry.
    // TODO(aroman) Consider using BigInt, but only if the V8 version supports
    // it. Check to see what V8 versions support BigInt.
    case tINT64:       return new Value(isolate, v8::Number::New(isolate, double(val.Int64))); break;
    case tOBJECT:      return new Value(isolate, v8::Object::New(isolate)); break;
    case tSTRING: {
      return new Value(isolate, v8::String::NewFromUtf8(
        isolate, val.Mem.ptr, v8::NewStringType::kNormal, val.Mem.len).ToLocalChecked());
      break;
    }
    case tUNDEFINED:   return new Value(isolate, v8::Undefined(isolate)); break;
  }
  return nullptr;
}

ValueTuple v8_Value_Get(ContextPtr ctxptr, PersistentValuePtr valueptr, const char* field) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return (ValueTuple){nullptr, 0, DupString("Not an object")};
  }

  // We can safely call `ToLocalChecked`, because
  // we've just created the local object above.
  v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();

  v8::Local<v8::Value> localValue = object->Get(ctx, v8::String::NewFromUtf8(isolate, field)).ToLocalChecked();

  return (ValueTuple){
    new Value(isolate, localValue),
    v8_Value_KindsFromLocal(localValue),
    nullptr,
  };
}

ValueTuple v8_Value_GetIdx(ContextPtr ctxptr, PersistentValuePtr valueptr, int idx) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return (ValueTuple){nullptr, 0, DupString("Not an object")};
  }

  v8::Local<v8::Value> obj;
  if (maybeObject->IsArrayBuffer()) {
    v8::ArrayBuffer* bufPtr = v8::ArrayBuffer::Cast(*maybeObject);
    if (idx < bufPtr->GetContents().ByteLength()) {
      obj = v8::Number::New(isolate, ((unsigned char*)bufPtr->GetContents().Data())[idx]);
    } else {
      obj = v8::Undefined(isolate);
    }
  } else {
    // We can safely call `ToLocalChecked`, because
    // we've just created the local object above.
    v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();
    obj = object->Get(ctx, uint32_t(idx)).ToLocalChecked();
  }
  return (ValueTuple){new Value(isolate, obj), v8_Value_KindsFromLocal(obj), nullptr};
}

Error v8_Value_Set(ContextPtr ctxptr, PersistentValuePtr valueptr,
                   const char* field, PersistentValuePtr new_valueptr) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return DupString("Not an object");
  }

  // We can safely call `ToLocalChecked`, because
  // we've just created the local object above.
  v8::Local<v8::Object> object =
      maybeObject->ToObject(ctx).ToLocalChecked();

  Value* new_value = static_cast<Value*>(new_valueptr);
  v8::Local<v8::Value> new_value_local = new_value->Get(isolate);
  v8::Maybe<bool> res =
    object->Set(ctx, v8::String::NewFromUtf8(isolate, field), new_value_local);

  if (res.IsNothing()) {
    return DupString("Something went wrong -- set returned nothing.");
  } else if (!res.FromJust()) {
    return DupString("Something went wrong -- set failed.");
  }
  return (Error){nullptr, 0};
}

Error v8_Value_SetIdx(ContextPtr ctxptr, PersistentValuePtr valueptr,
                      int idx, PersistentValuePtr new_valueptr) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return DupString("Not an object");
  }

  Value* new_value = static_cast<Value*>(new_valueptr);
  v8::Local<v8::Value> new_value_local = new_value->Get(isolate);
  if (maybeObject->IsArrayBuffer()) {
    v8::ArrayBuffer* bufPtr = v8::ArrayBuffer::Cast(*maybeObject);
    if (!new_value_local->IsNumber()) {
      return DupString("Cannot assign non-number into array buffer");
    } else if (idx >= bufPtr->GetContents().ByteLength()) {
      return DupString("Cannot assign to an index beyond the size of an array buffer");
    } else {
      ((unsigned char*)bufPtr->GetContents().Data())[idx] = new_value_local->ToNumber(ctx).ToLocalChecked()->Value();
    }
  } else {
    // We can safely call `ToLocalChecked`, because
    // we've just created the local object above.
    v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();

    v8::Maybe<bool> res = object->Set(ctx, uint32_t(idx), new_value_local);

    if (res.IsNothing()) {
      return DupString("Something went wrong -- set returned nothing.");
    } else if (!res.FromJust()) {
      return DupString("Something went wrong -- set failed.");
    }
  }

  return (Error){nullptr, 0};
}

ValueTuple v8_Value_Call(ContextPtr ctxptr,
                             PersistentValuePtr funcptr,
                             PersistentValuePtr selfptr,
                             int argc, PersistentValuePtr* argvptr) {
  VALUE_SCOPE(ctxptr);

  v8::TryCatch try_catch(isolate);
  try_catch.SetVerbose(false);

  v8::Local<v8::Value> func_val = static_cast<Value*>(funcptr)->Get(isolate);
  if (!func_val->IsFunction()) {
    return (ValueTuple){nullptr, 0, DupString("Not a function")};
  }
  v8::Local<v8::Function> func = v8::Local<v8::Function>::Cast(func_val);

  v8::Local<v8::Value> self;
  if (selfptr == nullptr) {
    self = ctx->Global();
  } else {
    self = static_cast<Value*>(selfptr)->Get(isolate);
  }

  v8::Local<v8::Value>* argv = new v8::Local<v8::Value>[argc];
  for (int i = 0; i < argc; i++) {
    argv[i] = static_cast<Value*>(argvptr[i])->Get(isolate);
  }

  v8::MaybeLocal<v8::Value> result = func->Call(ctx, self, argc, argv);

  delete[] argv;

  if (result.IsEmpty()) {
    return (ValueTuple){nullptr, 0, DupString(report_exception(isolate, ctx, try_catch))};
  }

  v8::Local<v8::Value> value = result.ToLocalChecked();
  return (ValueTuple){
    static_cast<PersistentValuePtr>(new Value(isolate, value)),
    v8_Value_KindsFromLocal(value),
    nullptr
  };
}

ValueTuple v8_Value_New(ContextPtr ctxptr,
                            PersistentValuePtr funcptr,
                            int argc, PersistentValuePtr* argvptr) {
  VALUE_SCOPE(ctxptr);

  v8::TryCatch try_catch(isolate);
  try_catch.SetVerbose(false);

  v8::Local<v8::Value> func_val = static_cast<Value*>(funcptr)->Get(isolate);
  if (!func_val->IsFunction()) {
    return (ValueTuple){nullptr, 0, DupString("Not a function")};
  }
  v8::Local<v8::Function> func = v8::Local<v8::Function>::Cast(func_val);

  v8::Local<v8::Value>* argv = new v8::Local<v8::Value>[argc];
  for (int i = 0; i < argc; i++) {
    argv[i] = static_cast<Value*>(argvptr[i])->Get(isolate);
  }

  v8::MaybeLocal<v8::Object> result = func->NewInstance(ctx, argc, argv);

  delete[] argv;

  if (result.IsEmpty()) {
    return (ValueTuple){nullptr, 0, DupString(report_exception(isolate, ctx, try_catch))};
  }

  v8::Local<v8::Value> value = result.ToLocalChecked();
  return (ValueTuple){
    static_cast<PersistentValuePtr>(new Value(isolate, value)),
    v8_Value_KindsFromLocal(value),
    nullptr
  };
}

void v8_Value_Release(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  if (valueptr == nullptr || ctxptr == nullptr)  {
    return;
  }

  ISOLATE_SCOPE(static_cast<Context*>(ctxptr)->isolate);

  Value* value = static_cast<Value*>(valueptr);
  value->Reset();
  delete value;
}

String v8_Value_String(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  VALUE_SCOPE(ctxptr);

  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);
  return DupString(value->ToString());
}

double v8_Value_Float64(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  VALUE_SCOPE(ctxptr);
  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);
  v8::Maybe<double> val = value->NumberValue(ctx);
  if (val.IsNothing()) {
    return 0;
  }
  return val.ToChecked();
}
int64_t v8_Value_Int64(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  VALUE_SCOPE(ctxptr);
  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);
  v8::Maybe<int64_t> val = value->IntegerValue(ctx);
  if (val.IsNothing()) {
    return 0;
  }
  return val.ToChecked();
}
int v8_Value_Bool(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  VALUE_SCOPE(ctxptr);
  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);
  v8::Maybe<bool> val = value->BooleanValue(ctx);
  if (val.IsNothing()) {
    return 0;
  }
  return val.ToChecked() ? 1 : 0;
}

ByteArray v8_Value_Bytes(ContextPtr ctxptr, PersistentValuePtr valueptr) {
  VALUE_SCOPE(ctxptr);

  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);

  v8::ArrayBuffer* bufPtr;

  if (value->IsTypedArray()) {
    bufPtr = *v8::TypedArray::Cast(*value)->Buffer();
  } else if (value->IsArrayBuffer()) {
    bufPtr = v8::ArrayBuffer::Cast(*value);
  } else {
    return (ByteArray){ nullptr, 0 };
  }

  if (bufPtr == NULL) {
    return (ByteArray){ nullptr, 0 };
  }

  return (ByteArray){
    static_cast<const char*>(bufPtr->GetContents().Data()),
    static_cast<int>(bufPtr->GetContents().ByteLength()),
  };
}

HeapStatistics v8_Isolate_GetHeapStatistics(IsolatePtr isolate_ptr) {
  if (isolate_ptr == nullptr) {
    return HeapStatistics{0};
  }
  ISOLATE_SCOPE(static_cast<v8::Isolate*>(isolate_ptr));
  v8::HeapStatistics hs;
  isolate->GetHeapStatistics(&hs);
  return HeapStatistics{
    hs.total_heap_size(),
    hs.total_heap_size_executable(),
    hs.total_physical_size(),
    hs.total_available_size(),
    hs.used_heap_size(),
    hs.heap_size_limit(),
    hs.malloced_memory(),
    hs.peak_malloced_memory(),
    hs.does_zap_garbage()
  };
}

void v8_Isolate_LowMemoryNotification(IsolatePtr isolate_ptr) {
  if (isolate_ptr == nullptr) {
    return;
  }
  ISOLATE_SCOPE(static_cast<v8::Isolate*>(isolate_ptr));
  isolate->LowMemoryNotification();
}

ValueTuple v8_Value_PromiseInfo(ContextPtr ctxptr, PersistentValuePtr valueptr,
                               int* promise_state) {
  VALUE_SCOPE(ctxptr);
  v8::Local<v8::Value> value = static_cast<Value*>(valueptr)->Get(isolate);
  if (!value->IsPromise()) { // just in case
    return (ValueTuple){nullptr, 0, DupString("Not a promise")};
  }

  v8::Promise* prom = v8::Promise::Cast(*value);
  *promise_state = prom->State();
  if (prom->State() == v8::Promise::PromiseState::kPending) {
    return (ValueTuple){nullptr, 0, nullptr};
  }
  v8::Local<v8::Value> res = prom->Result();
  return (ValueTuple){new Value(isolate, res), v8_Value_KindsFromLocal(res), nullptr};
}

} // extern "C"
