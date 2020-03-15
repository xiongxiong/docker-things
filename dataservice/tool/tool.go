package tool

import (
	"fmt"
	"log"

	"emperror.dev/errors"
)

// Error get error from any interface{}
func Error(x interface{}) (err error) {
	if x == nil {
		return
	}

	if er, ok := x.(error); ok {
		err = er
	} else {
		err = fmt.Errorf("%+v", x)
	}
	return
}

// CheckThenPanic check err, if nil print ok, else panic error.
func CheckThenPanic(err error, msg string) {
	if err != nil {
		panic(errors.WithStack(fmt.Errorf("%s <FAILURE> -- %s", msg, err)))
	} else {
		log.Printf("%s <SUCCESS>", msg)
	}
}

// CheckThenPrint check err, if nil print ok, else print error.
func CheckThenPrint(err error, msg string) {
	if err != nil {
		log.Printf("%s <FAILURE> -- %s", msg, err)
	} else {
		log.Printf("%s <SUCCESS>", msg)
	}
}

// ErrorThenPanic check error and panic
func ErrorThenPanic(err error, msg string) {
	if err != nil {
		panic(errors.WithStack(fmt.Errorf("%s -- %s", msg, err)))
	}
}

// ErrorThenPrint check error and print
func ErrorThenPrint(err error, msg string) {
	if err != nil {
		log.Printf("%s -- %s", msg, err)
	}
}
