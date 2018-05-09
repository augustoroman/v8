package v8

import "testing"

func BenchmarkGetValue(b *testing.B) {
	ctx := NewIsolate().NewContext()

	_, err := ctx.Eval(`var hello = "test"`, "bench.js")
	if err != nil {
		b.Fatal(err)
	}

	glob := ctx.Global()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := glob.Get("hello"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetNumberValue(b *testing.B) {
	ctx := NewIsolate().NewContext()
	val, err := ctx.Eval(`(157)`, "bench.js")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for n := 0; n < b.N; n += 2 {
		if res := val.Int64(); res != 157 {
			b.Fatal("Wrong value: ", res)
		}
		if res := val.Float64(); res != 157 {
			b.Fatal("Wrong value: ", res)
		}
	}
}

func BenchmarkContextCreate(b *testing.B) {
	ctx := NewIsolate().NewContext()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := ctx.Create(map[string]interface{}{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEval(b *testing.B) {
	iso := NewIsolate()
	ctx := iso.NewContext()

	script := `"hello"`

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := ctx.Eval(script, "bench-eval.js"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCallback(b *testing.B) {
	ctx := NewIsolate().NewContext()
	ctx.Global().Set("cb", ctx.Bind("cb", func(in CallbackArgs) (*Value, error) {
		return nil, nil
	}))

	script := `cb()`

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, err := ctx.Eval(script, "bench-cb.js"); err != nil {
			b.Fatal(err)
		}
	}
}
