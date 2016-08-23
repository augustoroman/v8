package v8console

import (
	"fmt"

	"github.com/augustoroman/v8"
)

const jsConsoleStub = `console = (function() {
    var stored = [];
    var exception = undefined;
    function flush(new_console) {
        stored.forEach(function(log) {
            new_console[log.type].apply(new_console, log.args);
        });
        return exception;
    };
    function catch_exception(e) {
        console.error('Failed to make snapshot:', e);
        exception = e;
    };
    return {
        __flush: flush,
        __catch: catch_exception,
        log:   function() { stored.push({type: 'log',   args: arguments}); },
        info:  function() { stored.push({type: 'info',  args: arguments}); },
        warn:  function() { stored.push({type: 'warn',  args: arguments}); },
        error: function() { stored.push({type: 'error', args: arguments}); },
    };
})();`

// WrapForSnapshot wraps the provided javascript code with a small, global
// console stub object that will record all console logs.  This is necessary
// when creating a snapshot for code that expects console.log to exist.  It also
// surrounds the jsCode with a try/catch that logs the error, since otherwise
// the snapshot will quietly fail.
func WrapForSnapshot(jsCode string) string {
	return fmt.Sprintf(`
        // Prefix with the console stub:
        %s
        try {
            %s
        } catch (e) {
            console.__catch(e); // Store and log the exception to error.
        }
    `, jsConsoleStub, jsCode)
}

// FlushSnapshotAndInject replaces the stub console operations with the console
// described by Config and flushes any stored log messages to the new console.
// This is specifically intended for adapting a Context created using
// WrapForSnapshot().
func FlushSnapshotAndInject(ctx *v8.Context, c Config) (exception *v8.Value) {
	// Store a reference to the previous console code for flushing any stored
	// log messages (see end of the func).  This should never fail to return
	// a *v8.Value, even if it's "undefined".
	previous, err := ctx.Global().Get("console")
	if err != nil || previous == nil {
		panic(fmt.Errorf("Global() must be an object: %v", err))
	}

	// Inject the new Console.
	c.Inject(ctx)
	// Get the new console object.  This should never fail to return a
	// *v8.Value, even if it's "undefined".  However, after the injection
	// above it should be the console object we just injected.
	current, err := ctx.Global().Get("console")
	if err != nil || current == nil {
		panic(fmt.Errorf("Global() must be an object: %v", err))
	}

	// Now flush any logs stored by the snapshot code above.  If the snapshot
	// code was not used, this will attempt to make a few calls and will fail
	// with no bad side-effects.

	// However this may fail since "undefined" won't allow .Get() at all. Even
	// if previous is an Object, it may return "undefined" if the previous
	// console object didn't have __flush.
	flush, err := previous.Get("__flush")
	if err != nil || flush == nil {
		return nil
	}

	// If flush is "undefined", this will fail with err != nil.  That's ok.
	// If it works, we flushed any stored logs to the new Console.
	// Otherwise, nothing happens.
	exception, err = flush.Call(previous, current)
	if err != nil || exception == nil {
		return nil
	}

	// Finally, check the returned exception value.  It's probably "undefined",
	// in which case we didn't have any error and we should return nil:
	if exception.String() == "undefined" {
		return nil
	}

	// Uh oh.  Looks like we actually got an exception.
	return exception
}
