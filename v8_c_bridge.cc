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


extern "C" ValueErrorPair go_callback_handler(
    String id, CallerInfo info, int argc, PersistentValuePtr* argv);

class ArrayBufferAllocator : public v8::ArrayBuffer::Allocator {
 public:
  virtual void* Allocate(size_t length);
  virtual void* AllocateUninitialized(size_t length);
  virtual void Free(void* data, size_t);
};
void* ArrayBufferAllocator::Allocate(size_t length) {
  void* data = AllocateUninitialized(length);
  return data == nullptr ? data : memset(data, 0, length);
}
void* ArrayBufferAllocator::AllocateUninitialized(size_t length) { return malloc(length); }
void ArrayBufferAllocator::Free(void* data, size_t) { free(data); }

// We only need one, it's stateless.
ArrayBufferAllocator allocator;

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

std::string str(v8::Local<v8::Value> value) {
  v8::String::Utf8Value s(value);
  if (s.length() == 0) {
    return "";
  }
  return *s;
}

std::string report_exception(v8::Isolate* isolate, v8::TryCatch& try_catch) {
  std::stringstream ss;
  ss << "Uncaught exception: ";

  std::string exceptionStr = str(try_catch.Exception());
  ss << exceptionStr; // TODO(aroman) JSON-ify objects?

  if (!try_catch.Message().IsEmpty()) {
    if (!try_catch.Message()->GetScriptResourceName()->IsUndefined()) {
      ss << std::endl
         << "at " << str(try_catch.Message()->GetScriptResourceName()) << ":"
         << try_catch.Message()->GetLineNumber() << ":"
         << try_catch.Message()->GetStartColumn() << std::endl
         << "  " << str(try_catch.Message()->GetSourceLine()) << std::endl
         << "  ";
      int start = try_catch.Message()->GetStartColumn();
      int end = try_catch.Message()->GetEndColumn();
      for (int i = 0; i < start; i++) {
        ss << " ";
      }
      for (int i = start; i < end; i++) {
        ss << "^";
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
  v8::Platform *platform = v8::platform::CreateDefaultPlatform();
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
  create_params.array_buffer_allocator = &allocator;
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

  v8::V8::SetCaptureStackTraceForUncaughtExceptions(true);

  v8::Local<v8::ObjectTemplate> globals = v8::ObjectTemplate::New(isolate);

  Context* ctx = new Context;
  ctx->ptr.Reset(isolate, v8::Context::New(isolate, nullptr, globals));
  ctx->isolate = isolate;
  return static_cast<ContextPtr>(ctx);
}
void v8_Isolate_Terminate(IsolatePtr isolate_ptr) {
  v8::Isolate* isolate = static_cast<v8::Isolate*>(isolate_ptr);
  v8::V8::TerminateExecution(isolate);
}
void v8_Isolate_Release(IsolatePtr isolate_ptr) {
  if (isolate_ptr == nullptr) {
    return;
  }
  v8::Isolate* isolate = static_cast<v8::Isolate*>(isolate_ptr);
  isolate->Dispose();
}

ValueErrorPair v8_Context_Run(ContextPtr ctxptr, const char* code, const char* filename) {
  Context* ctx = static_cast<Context*>(ctxptr);
  v8::Isolate* isolate = ctx->isolate;
  v8::Locker locker(isolate);
  v8::Isolate::Scope isolate_scope(isolate);
  v8::HandleScope handle_scope(isolate);
  v8::Context::Scope context_scope(ctx->ptr.Get(isolate));
  v8::TryCatch try_catch;
  try_catch.SetVerbose(false);

  filename = filename ? filename : "(no file)";

  ValueErrorPair res = { nullptr, nullptr };

  v8::Local<v8::Script> script = v8::Script::Compile(
      v8::String::NewFromUtf8(isolate, code),
      v8::String::NewFromUtf8(isolate, filename));

  if (script.IsEmpty()) {
    res.error_msg = DupString(report_exception(isolate, try_catch));
    return res;
  }

  v8::Local<v8::Value> result = script->Run();

  if (result.IsEmpty()) {
    res.error_msg = DupString(report_exception(isolate, try_catch));
  } else {
    res.Value = static_cast<PersistentValuePtr>(new Value(isolate, result));
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
  PersistentValuePtr argv[argc];
  for (int i = 0; i < argc; i++) {
    argv[i] = new Value(iso, args[i]);
  }

  ValueErrorPair result =
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
    case tSTRING:
      return new Value(isolate, v8::String::NewFromUtf8(
        isolate, val.Str.ptr, v8::NewStringType::kNormal, val.Str.len).ToLocalChecked());
    case tNUMBER:    return new Value(isolate, v8::Number::New(isolate, val.Num));           break;
    case tBOOL:      return new Value(isolate, v8::Boolean::New(isolate, val.BoolVal == 1)); break;
    case tOBJECT:    return new Value(isolate, v8::Object::New(isolate));                    break;
    case tARRAY:     return new Value(isolate, v8::Array::New(isolate, val.Len));            break;
    case tUNDEFINED: return new Value(isolate, v8::Undefined(isolate));                      break;
  }
  return nullptr;
}

ValueErrorPair v8_Value_Get(ContextPtr ctxptr, PersistentValuePtr valueptr, const char* field) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return (ValueErrorPair){nullptr, DupString("Not an object")};
  }

  // We can safely call `ToLocalChecked`, because
  // we've just created the local object above.
  v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();

  ValueErrorPair res = { nullptr, nullptr };
  res.Value = new Value(isolate,
    object->Get(ctx, v8::String::NewFromUtf8(isolate, field)).ToLocalChecked());
  return res;
}

ValueErrorPair v8_Value_GetIdx(ContextPtr ctxptr, PersistentValuePtr valueptr, int idx) {
  VALUE_SCOPE(ctxptr);

  Value* value = static_cast<Value*>(valueptr);
  v8::Local<v8::Value> maybeObject = value->Get(isolate);
  if (!maybeObject->IsObject()) {
    return (ValueErrorPair){nullptr, DupString("Not an object")};
  }

  // We can safely call `ToLocalChecked`, because
  // we've just created the local object above.
  v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();

  ValueErrorPair res = { nullptr, nullptr };
  res.Value = new Value(isolate, object->Get(ctx, uint32_t(idx)).ToLocalChecked());
  return res;
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

  // We can safely call `ToLocalChecked`, because
  // we've just created the local object above.
  v8::Local<v8::Object> object = maybeObject->ToObject(ctx).ToLocalChecked();


  Value* new_value = static_cast<Value*>(new_valueptr);
  v8::Local<v8::Value> new_value_local = new_value->Get(isolate);
  v8::Maybe<bool> res = object->Set(ctx, uint32_t(idx), new_value_local);

  if (res.IsNothing()) {
    return DupString("Something went wrong -- set returned nothing.");
  } else if (!res.FromJust()) {
    return DupString("Something went wrong -- set failed.");
  }

  return (Error){nullptr, 0};
}

ValueErrorPair v8_Value_Call(ContextPtr ctxptr,
                             PersistentValuePtr funcptr,
                             PersistentValuePtr selfptr,
                             int argc, PersistentValuePtr* argvptr) {
  VALUE_SCOPE(ctxptr);

  v8::TryCatch try_catch;
  try_catch.SetVerbose(false);

  v8::Local<v8::Value> func_val = static_cast<Value*>(funcptr)->Get(isolate);
  if (!func_val->IsFunction()) {
    return (ValueErrorPair){nullptr, DupString("Not a function")};
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
    return (ValueErrorPair){nullptr, DupString(report_exception(isolate, try_catch))};
  }

  return (ValueErrorPair){
    static_cast<PersistentValuePtr>(new Value(isolate, result.ToLocalChecked())),
    nullptr
  };
}

ValueErrorPair v8_Value_New(ContextPtr ctxptr,
                            PersistentValuePtr funcptr,
                            int argc, PersistentValuePtr* argvptr) {
  VALUE_SCOPE(ctxptr);

  v8::TryCatch try_catch;
  try_catch.SetVerbose(false);

  v8::Local<v8::Value> func_val = static_cast<Value*>(funcptr)->Get(isolate);
  if (!func_val->IsFunction()) {
    return (ValueErrorPair){nullptr, DupString("Not a function")};
  }
  v8::Local<v8::Function> func = v8::Local<v8::Function>::Cast(func_val);

  v8::Local<v8::Value>* argv = new v8::Local<v8::Value>[argc];
  for (int i = 0; i < argc; i++) {
    argv[i] = static_cast<Value*>(argvptr[i])->Get(isolate);
  }

  v8::MaybeLocal<v8::Object> result = func->NewInstance(ctx, argc, argv);

  delete[] argv;

  if (result.IsEmpty()) {
    return (ValueErrorPair){nullptr, DupString(report_exception(isolate, try_catch))};
  }

  return (ValueErrorPair){
    static_cast<PersistentValuePtr>(new Value(isolate, result.ToLocalChecked())),
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


} // extern "C"
