package main

type testStruct struct {
	embName
	Deleted bool `gogen-dump:"-"`
	Hm      struct {
		Balance *[3]**int16
		Hm      struct {
			AccountAge int
			Any        []interface{} `gogen-dump:"*embName []embName []*embName []*float32"`
		}
		Foo [][2]map[rune]***[]*int16
	}
	DingDong struct {
		R         rune
		FixedSize [9][7]float64
	}
	Age ***uint
}

type embName struct {
	FirstName   string
	MiddleNames *[]***[5]*string
	LastName    **string
	Fn          func()
	Ch          chan bool
}
