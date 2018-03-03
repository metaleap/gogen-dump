package main

type testStruct struct {
	embName
	Deleted    bool
	Balance    complex128
	AccountAge float64
	Age        byte
	R          rune
}

type embName struct {
	FirstName   string
	MiddleNames []string
	LastName    string
}
