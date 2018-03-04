package main

type testStruct struct {
	embName
	Deleted    bool
	Balance    *[3]**int16
	AccountAge int
	Any        []interface{} `gendump:"*embName []embName []*embName []*float32"`
	Foo        map[string]complex128
	Age        ***uint
	R          rune
}

type embName struct {
	FirstName   string
	MiddleNames *[]***[4]*string
	LastName    **string
}
