package kube

import "reflect"

func GetFields(r interface{}) []string {
	var fields []string

	s := reflect.ValueOf(r).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		fields = append(fields, typeOfT.Field(i).Name)
	}
	return fields
}

func GetField(r interface{}, field string) interface{} {
	v := reflect.ValueOf(r)
	f := reflect.Indirect(v).FieldByName(field)
	return f.Interface()
}
