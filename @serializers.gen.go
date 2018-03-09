package main

// Code generated by github.com/metaleap/gogen-dump — DO NOT EDIT.

import (
	"bytes"
	"io"
	"unsafe"

	fmt "fmt"
)

func (me *fixed) writeTo(buf *bytes.Buffer) (err error) {

	buf.Write((*[2036]byte)(unsafe.Pointer(me))[:])

	return
}

func (me *fixed) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := me.writeTo(&buf); err != nil {
		return 0, err
	}
	return buf.WriteTo(w)
}

func (me *fixed) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *fixed) ReadFrom(r io.Reader) (n int64, err error) {
	var buf bytes.Buffer
	if n, err = buf.ReadFrom(r); err == nil {
		err = me.UnmarshalBinary(buf.Bytes())
	}
	return
}

func (me *fixed) UnmarshalBinary(data []byte) (err error) {

	*me = *((*fixed)(unsafe.Pointer(&data[0])))

	return
}

func (me *testStruct) writeTo(buf *bytes.Buffer) (err error) {

	var data bytes.Buffer

	if err = me.embName.writeTo(&data); err != nil {
		return
	}
	lembName := (data.Len())
	buf.Write((*[8]byte)(unsafe.Pointer(&lembName))[:])
	data.WriteTo(buf)

	buf.Write(((*[16]byte)(unsafe.Pointer(&(me.DingDong.Complex))))[:])

	buf.Write(((*[504]byte)(unsafe.Pointer(&(me.DingDong.FixedSize[0]))))[:])

	if me.Hm.Balance == nil {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
		for i0 := 0; i0 < 3; i0++ {
			if (*me.Hm.Balance)[i0] == nil {
				buf.WriteByte(0)
			} else {
				buf.WriteByte(1)
				if *(*me.Hm.Balance)[i0] == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					buf.Write(((*[2]byte)(unsafe.Pointer(*(*me.Hm.Balance)[i0])))[:])
				}
			}
		}
	}

	buf.Write(((*[8]byte)(unsafe.Pointer(&(me.Hm.Hm.AccountAge))))[:])

	lHmHmLookie := (len(me.Hm.Hm.Lookie))
	buf.Write((*[8]byte)(unsafe.Pointer(&lHmHmLookie))[:])
	if (lHmHmLookie) > 0 {
		buf.Write((*[1125899906842623]byte)(unsafe.Pointer(&me.Hm.Hm.Lookie[0]))[:2036*(lHmHmLookie)])
	}

	lHmHmAny := (len(me.Hm.Hm.Any))
	buf.Write((*[8]byte)(unsafe.Pointer(&lHmHmAny))[:])
	for k0, m0 := range me.Hm.Hm.Any {
		if k0 == nil {
			buf.WriteByte(0)
		} else {
			buf.WriteByte(1)
			buf.Write((*[2036]byte)(unsafe.Pointer(k0))[:])
		}
		{
			switch t := m0.(type) {
			case fixed:
				buf.WriteByte(1)
				buf.Write((*[2036]byte)(unsafe.Pointer(&t))[:])
			case *fixed:
				buf.WriteByte(2)
				if t == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					buf.Write((*[2036]byte)(unsafe.Pointer(t))[:])
				}
			case []fixed:
				buf.WriteByte(3)
				lm0 := (len(t))
				buf.Write((*[8]byte)(unsafe.Pointer(&lm0))[:])
				if (lm0) > 0 {
					buf.Write((*[1125899906842623]byte)(unsafe.Pointer(&t[0]))[:2036*(lm0)])
				}
			case [5][6]fixed:
				buf.WriteByte(4)
				buf.Write(((*[61080]byte)(unsafe.Pointer(&(t[0]))))[:])
			case *embName:
				buf.WriteByte(5)
				if t == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					if err = t.writeTo(&data); err != nil {
						return
					}
					lm0 := (data.Len())
					buf.Write((*[8]byte)(unsafe.Pointer(&lm0))[:])
					data.WriteTo(buf)
				}
			case []embName:
				buf.WriteByte(6)
				lm0 := (len(t))
				buf.Write((*[8]byte)(unsafe.Pointer(&lm0))[:])
				for i1 := 0; i1 < (lm0); i1++ {
					if err = t[i1].writeTo(&data); err != nil {
						return
					}
					li1 := (data.Len())
					buf.Write((*[8]byte)(unsafe.Pointer(&li1))[:])
					data.WriteTo(buf)
				}
			case []*embName:
				buf.WriteByte(7)
				lm0 := (len(t))
				buf.Write((*[8]byte)(unsafe.Pointer(&lm0))[:])
				for i1 := 0; i1 < (lm0); i1++ {
					if t[i1] == nil {
						buf.WriteByte(0)
					} else {
						buf.WriteByte(1)
						if err = t[i1].writeTo(&data); err != nil {
							return
						}
						li1 := (data.Len())
						buf.Write((*[8]byte)(unsafe.Pointer(&li1))[:])
						data.WriteTo(buf)
					}
				}
			case []*float32:
				buf.WriteByte(8)
				lm0 := (len(t))
				buf.Write((*[8]byte)(unsafe.Pointer(&lm0))[:])
				for i1 := 0; i1 < (lm0); i1++ {
					if t[i1] == nil {
						buf.WriteByte(0)
					} else {
						buf.WriteByte(1)
						buf.Write(((*[4]byte)(unsafe.Pointer(t[i1])))[:])
					}
				}
			case nil:
				buf.WriteByte(0)
			default:
				return fmt.Errorf("testStruct.m0: type %T not mentioned in tagged-union field-tag", t)
			}
		}
	}

	lHmFoo := (len(me.Hm.Foo))
	buf.Write((*[8]byte)(unsafe.Pointer(&lHmFoo))[:])
	for i0 := 0; i0 < (lHmFoo); i0++ {
		for i1 := 0; i1 < 2; i1++ {
			li1 := (len(me.Hm.Foo[i0][i1]))
			buf.Write((*[8]byte)(unsafe.Pointer(&li1))[:])
			for k2, m2 := range me.Hm.Foo[i0][i1] {
				buf.Write(((*[4]byte)(unsafe.Pointer(&(k2))))[:])
				if m2 == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					if *m2 == nil {
						buf.WriteByte(0)
					} else {
						buf.WriteByte(1)
						if **m2 == nil {
							buf.WriteByte(0)
						} else {
							buf.WriteByte(1)
							lm2 := (len((***m2)))
							buf.Write((*[8]byte)(unsafe.Pointer(&lm2))[:])
							for i3 := 0; i3 < (lm2); i3++ {
								if (***m2)[i3] == nil {
									buf.WriteByte(0)
								} else {
									buf.WriteByte(1)
									buf.Write(((*[2]byte)(unsafe.Pointer((***m2)[i3])))[:])
								}
							}
						}
					}
				}
			}
		}
	}

	if me.Age == nil {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
		if *me.Age == nil {
			buf.WriteByte(0)
		} else {
			buf.WriteByte(1)
			if **me.Age == nil {
				buf.WriteByte(0)
			} else {
				buf.WriteByte(1)
				if ***me.Age == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					buf.Write(((*[8]byte)(unsafe.Pointer(***me.Age)))[:])
				}
			}
		}
	}

	return
}

func (me *testStruct) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := me.writeTo(&buf); err != nil {
		return 0, err
	}
	return buf.WriteTo(w)
}

func (me *testStruct) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *testStruct) ReadFrom(r io.Reader) (n int64, err error) {
	var buf bytes.Buffer
	if n, err = buf.ReadFrom(r); err == nil {
		err = me.UnmarshalBinary(buf.Bytes())
	}
	return
}

func (me *testStruct) UnmarshalBinary(data []byte) (err error) {

	var p int

	lembName := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	if err = me.embName.UnmarshalBinary(data[p : p+lembName]); err != nil {
		return
	}
	p += lembName

	me.DingDong.Complex = *((*complex128)(unsafe.Pointer(&data[p])))
	p += 16

	me.DingDong.FixedSize = *((*[9][7]float64)(unsafe.Pointer(&data[p])))
	p += 504

	{
		var p00 *[3]**int16
		if p++; data[p-1] != 0 {
			v10 := [3]**int16{}
			for i0 := 0; i0 < 3; i0++ {
				{
					var p01 **int16
					var p11 *int16
					if p++; data[p-1] != 0 {
						if p++; data[p-1] != 0 {
							v21 := *((*int16)(unsafe.Pointer(&data[p])))
							p += 2
							p11 = &v21
						}
						p01 = &p11
					}
					v10[i0] = p01
				}
			}
			p00 = &v10
		}
		me.Hm.Balance = p00
	}

	me.Hm.Hm.AccountAge = *((*int)(unsafe.Pointer(&data[p])))
	p += 8

	lHmHmLookie := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	me.Hm.Hm.Lookie = make([]fixed, lHmHmLookie)
	if (lHmHmLookie) > 0 {
		copy(((*[1125899906842623]byte)(unsafe.Pointer(&me.Hm.Hm.Lookie[0])))[0:2036*(lHmHmLookie)], data[p:p+(2036*(lHmHmLookie))])
		p += (2036 * (lHmHmLookie))
	}

	lHmHmAny := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	me.Hm.Hm.Any = make(map[*fixed]iface1, lHmHmAny)
	for i0 := 0; i0 < (lHmHmAny); i0++ {
		var bk0 *fixed
		var bm0 iface1
		{
			var p01 *fixed
			if p++; data[p-1] != 0 {
				v11 := *((*fixed)(unsafe.Pointer(&data[p])))
				p += 2036
				p01 = &v11
			}
			bk0 = p01
		}
		{
			t := data[p]
			p++
			switch t {
			case 1:
				var u fixed
				u = *((*fixed)(unsafe.Pointer(&data[p])))
				p += 2036
				bm0 = u
			case 2:
				var u *fixed
				{
					var p01 *fixed
					if p++; data[p-1] != 0 {
						v11 := *((*fixed)(unsafe.Pointer(&data[p])))
						p += 2036
						p01 = &v11
					}
					u = p01
				}
				bm0 = u
			case 3:
				var u []fixed
				lm0 := (*((*int)(unsafe.Pointer(&data[p]))))
				p += 8
				u = make([]fixed, lm0)
				if (lm0) > 0 {
					copy(((*[1125899906842623]byte)(unsafe.Pointer(&u[0])))[0:2036*(lm0)], data[p:p+(2036*(lm0))])
					p += (2036 * (lm0))
				}
				bm0 = u
			case 4:
				var u [5][6]fixed
				u = *((*[5][6]fixed)(unsafe.Pointer(&data[p])))
				p += 61080
				bm0 = u
			case 5:
				var u *embName
				{
					var p01 *embName
					if p++; data[p-1] != 0 {
						v11 := embName{}
						lm0 := (*((*int)(unsafe.Pointer(&data[p]))))
						p += 8
						if err = v11.UnmarshalBinary(data[p : p+lm0]); err != nil {
							return
						}
						p += lm0
						p01 = &v11
					}
					u = p01
				}
				bm0 = u
			case 6:
				var u []embName
				lm0 := (*((*int)(unsafe.Pointer(&data[p]))))
				p += 8
				u = make([]embName, lm0)
				for i1 := 0; i1 < (lm0); i1++ {
					li1 := (*((*int)(unsafe.Pointer(&data[p]))))
					p += 8
					if err = u[i1].UnmarshalBinary(data[p : p+li1]); err != nil {
						return
					}
					p += li1
				}
				bm0 = u
			case 7:
				var u []*embName
				lm0 := (*((*int)(unsafe.Pointer(&data[p]))))
				p += 8
				u = make([]*embName, lm0)
				for i1 := 0; i1 < (lm0); i1++ {
					{
						var p02 *embName
						if p++; data[p-1] != 0 {
							v12 := embName{}
							li1 := (*((*int)(unsafe.Pointer(&data[p]))))
							p += 8
							if err = v12.UnmarshalBinary(data[p : p+li1]); err != nil {
								return
							}
							p += li1
							p02 = &v12
						}
						u[i1] = p02
					}
				}
				bm0 = u
			case 8:
				var u []*float32
				lm0 := (*((*int)(unsafe.Pointer(&data[p]))))
				p += 8
				u = make([]*float32, lm0)
				for i1 := 0; i1 < (lm0); i1++ {
					{
						var p02 *float32
						if p++; data[p-1] != 0 {
							v12 := *((*float32)(unsafe.Pointer(&data[p])))
							p += 4
							p02 = &v12
						}
						u[i1] = p02
					}
				}
				bm0 = u
			default:
				bm0 = nil
			}
		}
		me.Hm.Hm.Any[bk0] = bm0
	}

	lHmFoo := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	me.Hm.Foo = make([][2]map[rune]***[]*int16, lHmFoo)
	for i0 := 0; i0 < (lHmFoo); i0++ {
		for i1 := 0; i1 < 2; i1++ {
			li1 := (*((*int)(unsafe.Pointer(&data[p]))))
			p += 8
			me.Hm.Foo[i0][i1] = make(map[rune]***[]*int16, li1)
			for i2 := 0; i2 < (li1); i2++ {
				var bk2 rune
				var bm2 ***[]*int16
				bk2 = *((*rune)(unsafe.Pointer(&data[p])))
				p += 4
				{
					var p03 ***[]*int16
					var p13 **[]*int16
					var p23 *[]*int16
					if p++; data[p-1] != 0 {
						if p++; data[p-1] != 0 {
							if p++; data[p-1] != 0 {
								lm2 := (*((*int)(unsafe.Pointer(&data[p]))))
								p += 8
								v33 := make([]*int16, lm2)
								for i3 := 0; i3 < (lm2); i3++ {
									{
										var p04 *int16
										if p++; data[p-1] != 0 {
											v14 := *((*int16)(unsafe.Pointer(&data[p])))
											p += 2
											p04 = &v14
										}
										v33[i3] = p04
									}
								}
								p23 = &v33
							}
							p13 = &p23
						}
						p03 = &p13
					}
					bm2 = p03
				}
				me.Hm.Foo[i0][i1][bk2] = bm2
			}
		}
	}

	{
		var p00 ****uint
		var p10 ***uint
		var p20 **uint
		var p30 *uint
		if p++; data[p-1] != 0 {
			if p++; data[p-1] != 0 {
				if p++; data[p-1] != 0 {
					if p++; data[p-1] != 0 {
						v40 := *((*uint)(unsafe.Pointer(&data[p]))) /* p += 8 */
						p30 = &v40
					}
					p20 = &p30
				}
				p10 = &p20
			}
			p00 = &p10
		}
		me.Age = p00
	}

	return
}

func (me *embName) writeTo(buf *bytes.Buffer) (err error) {

	buf.Write(((*[24432]byte)(unsafe.Pointer(&(me.LeFix[0]))))[:])

	lFirstName := (len(me.FirstName))
	buf.Write((*[8]byte)(unsafe.Pointer(&lFirstName))[:])
	buf.WriteString(me.FirstName)

	lMiddleNames := (len(me.MiddleNames))
	buf.Write((*[8]byte)(unsafe.Pointer(&lMiddleNames))[:])
	for i0 := 0; i0 < (lMiddleNames); i0++ {
		if me.MiddleNames[i0] == nil {
			buf.WriteByte(0)
		} else {
			buf.WriteByte(1)
			if *me.MiddleNames[i0] == nil {
				buf.WriteByte(0)
			} else {
				buf.WriteByte(1)
				if **me.MiddleNames[i0] == nil {
					buf.WriteByte(0)
				} else {
					buf.WriteByte(1)
					for i1 := 0; i1 < 5; i1++ {
						if (***me.MiddleNames[i0])[i1] == nil {
							buf.WriteByte(0)
						} else {
							buf.WriteByte(1)
							li1 := (len((*(***me.MiddleNames[i0])[i1])))
							buf.Write((*[8]byte)(unsafe.Pointer(&li1))[:])
							buf.WriteString((*(***me.MiddleNames[i0])[i1]))
						}
					}
				}
			}
		}
	}

	if me.LastName == nil {
		buf.WriteByte(0)
	} else {
		buf.WriteByte(1)
		if *me.LastName == nil {
			buf.WriteByte(0)
		} else {
			buf.WriteByte(1)
			lLastName := (len((**me.LastName)))
			buf.Write((*[8]byte)(unsafe.Pointer(&lLastName))[:])
			buf.WriteString((**me.LastName))
		}
	}

	return
}

func (me *embName) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if err := me.writeTo(&buf); err != nil {
		return 0, err
	}
	return buf.WriteTo(w)
}

func (me *embName) MarshalBinary() (data []byte, err error) {
	var buf bytes.Buffer
	if err = me.writeTo(&buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (me *embName) ReadFrom(r io.Reader) (n int64, err error) {
	var buf bytes.Buffer
	if n, err = buf.ReadFrom(r); err == nil {
		err = me.UnmarshalBinary(buf.Bytes())
	}
	return
}

func (me *embName) UnmarshalBinary(data []byte) (err error) {

	var p int

	me.LeFix = *((*[3][4]fixed)(unsafe.Pointer(&data[p])))
	p += 24432

	lFirstName := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	me.FirstName = string(data[p : p+lFirstName])
	p += lFirstName

	lMiddleNames := (*((*int)(unsafe.Pointer(&data[p]))))
	p += 8
	me.MiddleNames = make([]***[5]*string, lMiddleNames)
	for i0 := 0; i0 < (lMiddleNames); i0++ {
		{
			var p01 ***[5]*string
			var p11 **[5]*string
			var p21 *[5]*string
			if p++; data[p-1] != 0 {
				if p++; data[p-1] != 0 {
					if p++; data[p-1] != 0 {
						v31 := [5]*string{}
						for i1 := 0; i1 < 5; i1++ {
							{
								var p02 *string
								if p++; data[p-1] != 0 {
									li1 := (*((*int)(unsafe.Pointer(&data[p]))))
									p += 8
									v12 := string(data[p : p+li1])
									p += li1
									p02 = &v12
								}
								v31[i1] = p02
							}
						}
						p21 = &v31
					}
					p11 = &p21
				}
				p01 = &p11
			}
			me.MiddleNames[i0] = p01
		}
	}

	{
		var p00 **string
		var p10 *string
		if p++; data[p-1] != 0 {
			if p++; data[p-1] != 0 {
				lLastName := (*((*int)(unsafe.Pointer(&data[p]))))
				p += 8
				v20 := string(data[p : p+lLastName]) /* p += lLastName */
				p10 = &v20
			}
			p00 = &p10
		}
		me.LastName = p00
	}

	return
}
