// Package enums builds the Go types and functions for an enum declaration.
//
// An enum keeps one variant's data. Callers use constructors and payload methods
// so the compiler can change its private fields without changing those calls.
// Payload methods return copies. The checker has already selected the storage
// layout and checked each source use before this package builds Go declarations.
package enums
