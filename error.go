package engineio

import "fmt"

type Error struct {
	Msg  string
	Type string
	Desc string
}

func (err *Error) Error() string {
	return fmt.Sprintf("{\"msg\":\"%s\", \"type\":\"%s\", \"desc\":\"%s\"}", err.Msg, err.Type, err.Desc)
}
