package main

type testStruct struct {
	embName
	Deleted    bool
	Balance    *[3]**int16
	AccountAge int
	Any        []interface{} `gendump:"bool byte string int float64 float32"`
	Foo        *embName
	Age        ***uint
	R          rune
}

type embName struct {
	FirstName   string
	MiddleNames *[]***[4]*string
	LastName    **string
}
