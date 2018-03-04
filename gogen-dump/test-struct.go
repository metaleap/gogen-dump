package main

import (
	"encoding"
)

type testStruct struct {
	embName
	Deleted    bool
	Balance    *[]**int16
	AccountAge int
	Any        interface{} `gendump:"bool byte string int float64 float32"`
	Marsh      interface {
		encoding.BinaryMarshaler
		encoding.BinaryUnmarshaler
	}
	Age ***uint
	R   rune
}

type embName struct {
	FirstName string
	MiddleNames/*[]**[7]**/ uint16
	LastName **string
}
