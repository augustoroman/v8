// Package v8 provides a Go API for the the V8 javascript engine.
//
// This allows running javascript within a go executable.  The bindings
// have been tested with v8 builds between 5.1.281.16 through 6.7.77.
//
// V8 provides two main concepts for managing javascript state: Isolates and
// Contexts.  An isolate represents a single-threaded javascript engine that
// can manage one or more contexts.  A context is a sandboxed javascript
// execution environment.
//
// Thus, if you have one isolate, you could safely execute independent code in
// many different contexts created in that isolate.  The code in the various
// contexts would not interfere with each other, however no more than one
// context would ever be executing at a given time.
//
// If you have multiple isolates, they may be executing in separate threads
// simultaneously.
//
// This work is based off of several existing libraries:
//   * https://github.com/fluxio/go-v8
//   * https://github.com/kingland/go-v8
//   * https://github.com/mattn/go-v8
package v8
