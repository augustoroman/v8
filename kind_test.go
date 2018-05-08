package v8

import (
	"testing"
)

// This hard-codes a handful of kind <-> string mappings to ensure that our
// kind enum and kind string array are matched up.
func TestKindString(t *testing.T) {
	testcases := []struct {
		kind Kind
		str  string
	}{
		{KindUndefined, "Undefined"},
		{KindNativeError, "NativeError"},
		{KindRegExp, "RegExp"},
		{KindWebAssemblyCompiledModule, "WebAssemblyCompiledModule"},

		// Verify that we have N kinds and they are stringified reasonably.
		{kNumKinds, "NoSuchKind:47"},
	}
	for _, test := range testcases {
		if test.kind.String() != test.str {
			t.Errorf("Expected kind %q (%d) to stringify to %q",
				test.kind, test.kind, test.str)
		}
	}
}
