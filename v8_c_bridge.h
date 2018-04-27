#include <stddef.h>

#ifndef V8_C_BRIDGE_H
#define V8_C_BRIDGE_H

#ifdef __cplusplus
extern "C" {
#endif

typedef void* IsolatePtr;
typedef void* ContextPtr;
typedef void* PersistentValuePtr;

typedef struct {
    const char* ptr;
    int len;
} String;

typedef String Error;
typedef String StartupData;

typedef struct {
    size_t total_heap_size;
    size_t total_heap_size_executable;
    size_t total_physical_size;
    size_t total_available_size;
    size_t used_heap_size;
    size_t heap_size_limit;
    size_t malloced_memory;
    size_t peak_malloced_memory;
    size_t does_zap_garbage;
} HeapStatistics;

typedef struct {
    PersistentValuePtr Value;
    Error error_msg;
} ValueErrorPair;

typedef struct {
    String Funcname;
    String Filename;
    int Line;
    int Column;
} CallerInfo;

typedef struct { int Major, Minor, Build, Patch; } Version;
extern Version version;

typedef unsigned int uint32_t;

// v8_init must be called once before anything else.
extern void v8_init();

extern StartupData v8_CreateSnapshotDataBlob(const char* js);

extern IsolatePtr v8_Isolate_New(StartupData data);
extern ContextPtr v8_Isolate_NewContext(IsolatePtr isolate);
extern void       v8_Isolate_Terminate(IsolatePtr isolate);
extern void       v8_Isolate_Release(IsolatePtr isolate);

extern HeapStatistics       v8_Isolate_GetHeapStatistics(IsolatePtr isolate);
extern void       v8_Isolate_LowMemoryNotification(IsolatePtr isolate);

extern ValueErrorPair     v8_Context_Run(ContextPtr ctx,
                                         const char* code, const char* filename);
extern PersistentValuePtr v8_Context_RegisterCallback(ContextPtr ctx,
                                                      const char* name, const char* id);
extern PersistentValuePtr v8_Context_Global(ContextPtr ctx);
extern void               v8_Context_Release(ContextPtr ctx);

typedef enum { tSTRING, tBOOL, tNUMBER, tOBJECT, tARRAY, tARRAYBUFFER, tUNDEFINED } ImmediateValueType;
typedef struct {
    ImmediateValueType Type;
    String Str;
    int BoolVal;
    double Num;
    unsigned char* Bytes;
    int Len;
} ImmediateValue;
extern PersistentValuePtr v8_Context_Create(ContextPtr ctx, ImmediateValue val);

extern ValueErrorPair  v8_Value_Get(ContextPtr ctx, PersistentValuePtr value, const char* field);
extern Error           v8_Value_Set(ContextPtr ctx, PersistentValuePtr value,
                                    const char* field, PersistentValuePtr new_value);
extern ValueErrorPair  v8_Value_GetIdx(ContextPtr ctx, PersistentValuePtr value, int idx);
extern Error           v8_Value_SetIdx(ContextPtr ctx, PersistentValuePtr value,
                                       int idx, PersistentValuePtr new_value);
extern ValueErrorPair  v8_Value_PromiseResult(ContextPtr ctx, PersistentValuePtr value);
extern ValueErrorPair  v8_Value_Call(ContextPtr ctx,
                                     PersistentValuePtr func,
                                     PersistentValuePtr self,
                                     int argc, PersistentValuePtr* argv);
extern ValueErrorPair  v8_Value_New(ContextPtr ctx,
                                    PersistentValuePtr func,
                                    int argc, PersistentValuePtr* argv);
extern void   v8_Value_Release(ContextPtr ctx, PersistentValuePtr value);
extern String v8_Value_String(ContextPtr ctx, PersistentValuePtr value);
extern unsigned char* v8_Value_Bytes(ContextPtr ctx, PersistentValuePtr value, int * length);

#ifdef __cplusplus
}
#endif

#endif  // !defined(V8_C_BRIDGE_H)
