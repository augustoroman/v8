package v8

// #include <stdlib.h>
// #include <string.h>
// #include "v8_c_bridge.h"
// #cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -fno-rtti -std=c++11
// #cgo LDFLAGS: -pthread -L${SRCDIR}/libv8 -lv8_base -lv8_init -lv8_initializers -lv8_libbase -lv8_libplatform -lv8_libsampler -lv8_nosnapshot
import "C"

type promise struct {
	value
}

func (p *promise) Result() (Value, error) {
	return p.ctx.split(C.v8_Value_PromiseResult(p.ctx.ptr, p.ptr))
}
