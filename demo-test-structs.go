package main

type sixteen = complex128

type fixed struct {
	eight1 float64
	eight2 [1]uint64
	eight3 [2]int64
	eight4 [3]complex64
	four1  [4]float32
	four2  [5]int32
	four3  [6]uint32
	four4  [7]rune
	one1   [8]uint8
	one2   [9]int8
	one3   [10]byte
	sixt1  [11]complex128
	sixt2  [12]sixteen
}

type testStruct struct {
	embName
	Deleted bool `gogen-dump:"-"`
	Hm      struct {
		Balance *[3]**int16
		Hm      struct {
			AccountAge int
			Lookie     []*fixed
			Any        []interface{} `gogen-dump:"*embName []embName []*embName []*float32"`
		}
		Foo [][2]map[rune]***[]*int16
	}
	DingDong struct {
		Complex   complex128
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
