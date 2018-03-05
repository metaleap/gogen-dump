package main

type testStruct struct {
	embName
	Deleted    bool `gogen-dump:"-"`
	Balance    *[3]**int16
	AccountAge int
	Any        []interface{} `gogen-dump:"*embName []embName []*embName []*float32"`
	Foo        [][2]map[rune]***[]*int16
	Age        ***uint
	R          rune
	By         byte
}

type embName struct {
	FirstName   string
	MiddleNames *[]***[4]*string
	LastName    **string
	Fn          func()
	Ch          chan bool
}
