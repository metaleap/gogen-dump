package main

import (
	"time"
)

type iface1 interface{}

type iface2 = interface{}

type sixteen = complex128

type fixed struct {
	eight1 float64
	eight2 [1]uint64
	eight3 [2][3]int64
	eight4 [4][5]complex64
	four1  [6][7]float32
	four2  [8][9]int32
	four3  [8][7]uint32
	four4  [6][5]rune
	one1   [4][3]uint8
	one2   [2][1]int8
	one3   [2][3]byte
	sixt1  [4][5]complex128
	sixt2  [6][7]sixteen
}

type testStruct struct {
	embName
	Deleted  bool `gogen-dump:"-"`
	DingDong struct {
		Complex   complex128
		FixedSize [9][7]float64
	}
	Hm struct {
		Balance *[3]**int16
		Hm      struct {
			AccountAge int
			Lookie     [2]fixed
			HowLong    time.Duration `gogen-dump:"-"`
			When       time.Time
			Any        map[*fixed]iface1 `gogen-dump:"fixed *fixed []fixed [5][6]fixed *embName []embName []*embName []*float32"`
		}
		Foo [][2]map[rune]***[]*int16
	}
	Age ****uint
}

type embName struct {
	Fn          func()
	LeFix       [3][4]fixed
	FirstName   string
	MiddleNames []***[5]*string
	LastName    **string
	Ch          chan bool
}
