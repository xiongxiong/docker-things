package tool

import (
	"fmt"
	"log"
)

// Error get error from any interface{}
func Error(x interface{}) (err error) {
	if er, ok := x.(error); ok {
		err = er
	} else {
		err = fmt.Errorf("%+v", x)
	}
	return
}

// PanicError check error and panic
func PanicError(err error, msg string) {
	if err != nil {
		panic(fmt.Errorf("%s: %s", msg, err))
	}
}

// PrintError check error and print
func PrintError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}
