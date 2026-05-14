package logger

import (
	"go.uber.org/zap"
)

// These are aliases for Zap's field types.
var (
	Any        = zap.Any
	Array      = zap.Array
	Bool       = zap.Bool
	Complex64  = zap.Complex64
	Complex128 = zap.Complex128
	Dict       = zap.Dict
	Float64    = zap.Float64
	Float32    = zap.Float32
	Int        = zap.Int
	NamedError = zap.NamedError
	Namespace  = zap.Namespace
	Reflect    = zap.Reflect
	Skip       = zap.Skip
	Stack      = zap.Stack
	String     = zap.String
	Stringer   = zap.Stringer
	Time       = zap.Time
	Uint       = zap.Uint
	Object     = zap.Object
)
