package reflection

import "reflect"

func Register(v any) string { return reflect.TypeOf(v).String() }
