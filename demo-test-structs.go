package main

import (
	"sort"
	"time"
)

type gameWorld struct {
	Cities [123]city
}

type city struct {
	Name string
	// ClosestTo *city
	Companies []company
	Families  []family
	Schools   []school
}

type company struct {
	// Suppliers []*company
	// Clients   []*company
	Staff []*person
}

type school struct {
	Teachers []*person
	Pupils   []*person
}

type family struct {
	LastName string
	// Pets     map[string]petAnimal `ggd:"*petPiranha *petHamster *petCat *petDog"`
}

type person struct {
	FirstName string
	// Family      *family
	// DateOfBirth time.Time
	// Parents     [2]*person
	// FavPet      petAnimal `ggd:"*petPiranha *petHamster *petCat *petDog"`
	// Top5Hobbies [5]hobby
}

type hobby struct {
	Name      string
	Outdoorsy bool
	AvgPerDay struct {
		TimeNeededMinMax  [2]time.Duration
		CostInCentsMinMax [2]uint16
	}
	GroupSizeMinMax [2]uint8
}

type petAnimal interface {
	carnivore() bool
	mammal() bool
}

type pet struct {
	DailyFoodBill  float32
	AgeWhenAdopted time.Duration
	LastIllness    struct {
		Days          time.Duration
		Date          time.Time
		NotSerialized sort.Interface   `ggd:"-"`
		Notes         sort.StringSlice `ggd:"[]string"`
	}
	OrigCostIfKnown *complex128
}

func (me *pet) carnivore() bool { return true }
func (me *pet) mammal() bool    { return true }

type petPiranha struct {
	pet
	Weird map[*[2048]byte][]fixedSize
}

func (me *petPiranha) mammal() bool { return false }

type petHamster struct {
	pet
}

func (me *petHamster) carnivore() bool { return false }

type petCat struct {
	pet
	RabbitsSlaynPerDayOnAvg *uint8
}

type petDog struct {
	pet
	WalkLog map[time.Time][7]time.Duration
}

type fixedSize struct {
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
	sixt2  [6][7]complex384
}

type complex384 [3]complex128

/*
type iface1 interface{}

type iface2 = interface{}

type testStruct struct {
	embName
	Age       ****uint
	Deleted   bool
	subStruct struct {
		Complex   complex128
		FixedSize [9][7]float64
	}
	SkipThis bool `ggd:"-"`
	Hm       struct {
		Balance *[3]**int16
		Hm      struct {
			AccountAge int
			HowLong    [3]time.Duration // `ggd:"int64"` // not needed here because time.Duration is in tSynonyms by default for convenience
			Lookie     [2]fixed
			When       time.Time
			Any        map[*fixed]iface1 `ggd:"fixed *fixed []fixed [5][6]fixed *embName []embName []*embName []*float32"`
			Crikey     sort.StringSlice  `ggd:"[]string"`
		}
		Foo [][2]map[rune]***[]*int16
	}
}

type embName struct {
	Fn          func()
	LeFix       [3][4]fixed
	FirstName   string
	MiddleNames []***[5]*string
	LastName    **string
	TriState    ***bool
	Ch          chan bool
}
*/

type bytesBuffer struct{ b []byte }

func (me *bytesBuffer) bytes() []byte {
	return me.b
}

func (me *bytesBuffer) writeByte(b byte) {
	l, c := len(me.b), cap(me.b)
	if l == c {
		old := me.b
		me.b = make([]byte, l+1, c+c)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:l+1]
	}
	me.b[l] = b
}

func (me *bytesBuffer) write(b []byte) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, c+c)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}

func (me *bytesBuffer) writeString(b string) {
	l, c, n := len(me.b), cap(me.b), len(b)
	if ln := l + n; ln > c {
		old := me.b
		me.b = make([]byte, ln, c+c)
		copy(me.b[:l], old)
	} else {
		me.b = me.b[:ln]
	}
	copy(me.b[l:], b)
}
