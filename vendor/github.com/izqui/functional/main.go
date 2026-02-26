package functional

import "reflect"
import _ "fmt"

func Reduce(f interface{}, vs interface{}, in interface{}) interface{} {

	vf := reflect.ValueOf(f)
	vx := reflect.ValueOf(vs)

	l := vx.Len()

	a := reflect.New(reflect.TypeOf(in))
	v := a.Elem()

	v.Set(reflect.ValueOf(in))

	for i := 0; i < l; i++ {

		a := vf.Call([]reflect.Value{a.Elem(), vx.Index(i)})[0]
		v.Set(a)
	}

	return v.Interface()
}

func Filter(f interface{}, vs interface{}) interface{} {

	vf := reflect.ValueOf(f)
	vx := reflect.ValueOf(vs)

	l := vx.Len()

	tys := reflect.SliceOf(vf.Type().In(0))

	vss := []reflect.Value{}

	for i := 0; i < l; i++ {

		v := vx.Index(i)
		if vf.Call([]reflect.Value{v})[0].Bool() {

			vss = append(vss, v)
		}
	}

	vys := reflect.MakeSlice(tys, len(vss), len(vss))

	for i, v := range vss {
		vys.Index(i).Set(v)
	}

	return vys.Interface()
}

func Map(f interface{}, vs interface{}) interface{} {

	vf := reflect.ValueOf(f)
	vx := reflect.ValueOf(vs)

	l := vx.Len()

	tys := reflect.SliceOf(vf.Type().Out(0))
	vys := reflect.MakeSlice(tys, l, l)

	for i := 0; i < l; i++ {

		y := vf.Call([]reflect.Value{vx.Index(i)})[0]
		vys.Index(i).Set(y)
	}

	return vys.Interface()
}

/*
func Pipeline(fs interface{}, vs interface{}) interface{} {

	vfs := reflect.ValueOf(fs)
	vvs := reflect.ValueOf(vs)

	if vfs.Len() > 0 && vvs.Len() > 0 {

		l := vvs.Len()
		fl := vfs.Len()

		s := []interface{}{}
		fs := []interface{}{}

		for i := 0; i < l; i++ {

			s = append(s, vvs.Index(i))
		}

		for i := 0; i < fl; i++ {

			fs = append(fs, vfs.Index(i))
		}

		return Map(func(v interface{}) interface{} {

			return Reduce(func(a interface{}, f interface{}) interface{} {

				res := f.(reflect.Value).Call([]reflect.Value{a.(reflect.Value)})[0]

				fmt.Println("1", res)
				fmt.Println(reflect.Value(res))
				fmt.Println(res.Interface())

				return reflect.Value(res).Elem()

			}, fs, v)

		}, s)
	}

	return vs
}*/
