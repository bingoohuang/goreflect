package walk

import (
	"fmt"
	"reflect"
	"testing"
)

type TestEnterExitWalker struct {
	Locs []Location
}

func (t *TestEnterExitWalker) Enter(l Location) error {
	if t.Locs == nil {
		t.Locs = make([]Location, 0, 5)
	}

	t.Locs = append(t.Locs, l)
	return nil
}

func (t *TestEnterExitWalker) Exit(l Location) error {
	t.Locs = append(t.Locs, l)
	return nil
}

type TestPointerWalker struct {
	pointers []bool
	count    int
	enters   int
	exits    int
}

func (t *TestPointerWalker) PointerEnter(v bool) error {
	t.pointers = append(t.pointers, v)
	t.enters++
	if v {
		t.count++
	}
	return nil
}

func (t *TestPointerWalker) PointerExit(v bool) error {
	t.exits++
	if t.pointers[len(t.pointers)-1] != v {
		return fmt.Errorf("bad pointer exit '%t' at exit %d", v, t.exits)
	}
	t.pointers = t.pointers[:len(t.pointers)-1]
	return nil
}

type TestPrimitiveWalker struct {
	Value reflect.Value
}

func (t *TestPrimitiveWalker) Primitive(v reflect.Value) error {
	t.Value = v
	return nil
}

type TestPrimitiveCountWalker struct {
	Count int
}

func (t *TestPrimitiveCountWalker) Primitive(v reflect.Value) error {
	t.Count++
	return nil
}

type TestPrimitiveReplaceWalker struct {
	Value reflect.Value
}

func (t *TestPrimitiveReplaceWalker) Primitive(v reflect.Value) error {
	v.Set(reflect.ValueOf("bar"))
	return nil
}

type TestMapWalker struct {
	MapVal reflect.Value
	Keys   map[string]bool
	Values map[string]bool
}

func (t *TestMapWalker) Map(m reflect.Value) error {
	t.MapVal = m
	return nil
}

func (t *TestMapWalker) MapElem(m, k, v reflect.Value) error {
	if t.Keys == nil {
		t.Keys = make(map[string]bool)
		t.Values = make(map[string]bool)
	}

	t.Keys[k.Interface().(string)] = true
	t.Values[v.Interface().(string)] = true
	return nil
}

type TestMapElemReplaceWalker struct {
	ValueFn func(v reflect.Value) reflect.Value
}

func (t *TestMapElemReplaceWalker) Map(m reflect.Value) error {
	return nil
}

func (t *TestMapElemReplaceWalker) MapElem(m, k, v reflect.Value) error {
	m.SetMapIndex(k, t.ValueFn(v))
	return nil
}

type TestSliceWalker struct {
	Count    int
	SliceVal reflect.Value
}

func (t *TestSliceWalker) Slice(v reflect.Value) error {
	t.SliceVal = v
	return nil
}

func (t *TestSliceWalker) SliceElem(int, reflect.Value) error {
	t.Count++
	return nil
}

type TestStructWalker struct {
	Fields []string
}

func (t *TestStructWalker) Struct(v reflect.Value) error {
	return nil
}

func (t *TestStructWalker) StructField(sf reflect.StructField, v reflect.Value) error {
	if t.Fields == nil {
		t.Fields = make([]string, 0, 1)
	}

	t.Fields = append(t.Fields, sf.Name)
	return nil
}

func TestTestStructs(t *testing.T) {
	var raw interface{} = new(TestEnterExitWalker)
	if _, ok := raw.(EnterExitWalker); !ok {
		t.Fatal("EnterExitWalker is bad")
	}

	raw = new(TestPrimitiveWalker)
	if _, ok := raw.(PrimitiveWalker); !ok {
		t.Fatal("PrimitiveWalker is bad")
	}

	raw = new(TestMapWalker)
	if _, ok := raw.(MapWalker); !ok {
		t.Fatal("MapWalker is bad")
	}

	raw = new(TestSliceWalker)
	if _, ok := raw.(SliceWalker); !ok {
		t.Fatal("SliceWalker is bad")
	}

	raw = new(TestStructWalker)
	if _, ok := raw.(StructWalker); !ok {
		t.Fatal("StructWalker is bad")
	}
}

func TestWalk_Basic(t *testing.T) {
	w := new(TestPrimitiveWalker)

	type S struct {
		Foo string
	}

	data := &S{
		Foo: "foo",
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.Value.Kind() != reflect.String {
		t.Fatalf("bad: %#v", w.Value)
	}
}

func TestWalk_Basic_Replace(t *testing.T) {
	w := new(TestPrimitiveReplaceWalker)

	type S struct {
		Foo string
		Bar []interface{}
	}

	data := &S{
		Foo: "foo",
		Bar: []interface{}{[]string{"what"}},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if data.Foo != "bar" {
		t.Fatalf("bad: %#v", data.Foo)
	}
	if data.Bar[0].([]string)[0] != "bar" {
		t.Fatalf("bad: %#v", data.Bar)
	}
}

func TestWalk_Basic_ReplaceInterface(t *testing.T) {
	w := new(TestPrimitiveReplaceWalker)

	type S struct {
		Foo []interface{}
	}

	data := &S{
		Foo: []interface{}{"foo"},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestWalk_EnterExit(t *testing.T) {
	w := new(TestEnterExitWalker)

	type S struct {
		A string
		M map[string]string
	}

	data := &S{
		A: "foo",
		M: map[string]string{
			"a": "b",
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []Location{
		Loc,
		Struct,
		StructField,
		StructField,
		StructField,
		Map,
		MapKey,
		MapKey,
		MapValue,
		MapValue,
		Map,
		StructField,
		Struct,
		Loc,
	}
	if !reflect.DeepEqual(w.Locs, expected) {
		t.Fatalf("Bad: %#v", w.Locs)
	}
}

func TestWalk_Interface(t *testing.T) {
	w := new(TestPrimitiveCountWalker)

	type S struct {
		Foo string
		Bar []interface{}
	}

	var data interface{} = &S{
		Foo: "foo",
		Bar: []interface{}{[]string{"bar", "what"}, "baz"},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.Count != 4 {
		t.Fatalf("bad: %#v", w.Count)
	}
}

func TestWalk_Interface_nil(t *testing.T) {
	w := new(TestPrimitiveCountWalker)

	type S struct {
		Bar interface{}
	}

	var data interface{} = &S{}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestWalk_Map(t *testing.T) {
	w := new(TestMapWalker)

	type S struct {
		Foo map[string]string
	}

	data := &S{
		Foo: map[string]string{
			"foo": "foov",
			"bar": "barv",
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.MapVal.Interface(), data.Foo) {
		t.Fatalf("Bad: %#v", w.MapVal.Interface())
	}

	expectedK := map[string]bool{"foo": true, "bar": true}
	if !reflect.DeepEqual(w.Keys, expectedK) {
		t.Fatalf("Bad keys: %#v", w.Keys)
	}

	expectedV := map[string]bool{"foov": true, "barv": true}
	if !reflect.DeepEqual(w.Values, expectedV) {
		t.Fatalf("Bad values: %#v", w.Values)
	}
}

func TestWalk_Map_ReplaceValue(t *testing.T) {
	w := &TestMapElemReplaceWalker{
		ValueFn: func(v reflect.Value) reflect.Value {
			if v.Type().Kind() == reflect.String {
				return reflect.ValueOf("replaced")
			}

			if v.Type().Kind() == reflect.Interface {
				if elem := v.Elem(); elem.Type() == reflect.TypeOf(map[string]interface{}{}) {
					newMap := make(map[string]interface{})
					for _, k := range elem.MapKeys() {
						newMap[k.String()] = elem.MapIndex(k).Interface()
					}
					newMap["extra-replaced"] = "not-replaced"
					return reflect.ValueOf(newMap)
				} else if elem.Type().Kind() == reflect.String {
					return reflect.ValueOf("replaced")
				}
			}

			return v
		},
	}

	type S struct {
		Foo map[string]interface{}
	}

	data := &S{
		Foo: map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": map[string]string{"baz": "should-get-replaced"},
			},
		},
	}

	expected := &S{
		Foo: map[string]interface{}{
			"foo": map[string]interface{}{
				"bar":            map[string]string{"baz": "replaced"},
				"extra-replaced": "replaced",
			},
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if !reflect.DeepEqual(data, expected) {
		t.Fatalf("Values not equal: %#v", data)
	}
}

func TestWalk_Pointer(t *testing.T) {
	w := new(TestPointerWalker)

	type S struct {
		Foo string
		Bar *string
		Baz **string
	}

	s := ""
	sp := &s

	data := &S{
		Baz: &sp,
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.enters != 5 {
		t.Fatal("expected 4 values, saw", w.enters)
	}

	if w.count != 4 {
		t.Fatal("exptec 3 pointers, saw", w.count)
	}

	if w.exits != w.enters {
		t.Fatalf("number of enters (%d) and exits (%d) don't match", w.enters, w.exits)
	}
}

func TestWalk_PointerPointer(t *testing.T) {
	w := new(TestPointerWalker)

	s := ""
	sp := &s
	pp := &sp

	err := Walk(pp, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if w.enters != 2 {
		t.Fatal("expected 2 values, saw", w.enters)
	}

	if w.count != 2 {
		t.Fatal("expected 2 pointers, saw", w.count)
	}

	if w.exits != w.enters {
		t.Fatalf("number of enters (%d) and exits (%d) don't match", w.enters, w.exits)
	}
}

func TestWalk_Slice(t *testing.T) {
	w := new(TestSliceWalker)

	type S struct {
		Foo []string
	}

	data := &S{
		Foo: []string{"a", "b", "c"},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.SliceVal.Interface(), data.Foo) {
		t.Fatalf("bad: %#v", w.SliceVal.Interface())
	}

	if w.Count != 3 {
		t.Fatalf("Bad count: %d", w.Count)
	}
}

func TestWalk_SliceWithPtr(t *testing.T) {
	w := new(TestSliceWalker)

	// This is key, the panic only happened when the slice field was
	// an interface!
	type I interface{}

	type S struct {
		Foo []I
	}

	type Empty struct{}

	data := &S{
		Foo: []I{&Empty{}},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.SliceVal.Interface(), data.Foo) {
		t.Fatalf("bad: %#v", w.SliceVal.Interface())
	}

	if w.Count != 1 {
		t.Fatalf("Bad count: %d", w.Count)
	}
}

type testErr struct{}

func (t *testErr) Error() string {
	return "test error"
}

func TestWalk_Struct(t *testing.T) {
	w := new(TestStructWalker)

	// This makes sure we can also walk over pointer-to-pointers, and the ever
	// so rare pointer-to-interface
	type S struct {
		Foo string
		Bar *string
		Baz **string
		Err *error
	}

	bar := "ptr"
	baz := &bar
	e := error(&testErr{})

	data := &S{
		Foo: "foo",
		Bar: &bar,
		Baz: &baz,
		Err: &e,
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"Foo", "Bar", "Baz", "Err"}
	if !reflect.DeepEqual(w.Fields, expected) {
		t.Fatalf("bad: %#v", w.Fields)
	}
}

// Very similar to above test but used to fail for #2, copied here for
// regression testing
func TestWalk_StructWithPtr(t *testing.T) {
	w := new(TestStructWalker)

	type S struct {
		Foo string
		Bar string
		Baz *int
	}

	data := &S{
		Foo: "foo",
		Bar: "bar",
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := []string{"Foo", "Bar", "Baz"}
	if !reflect.DeepEqual(w.Fields, expected) {
		t.Fatalf("bad: %#v", w.Fields)
	}
}

type TestInterfaceMapWalker struct {
	MapVal reflect.Value
	Keys   map[string]bool
	Values map[interface{}]bool
}

func (t *TestInterfaceMapWalker) Map(m reflect.Value) error {
	t.MapVal = m
	return nil
}

func (t *TestInterfaceMapWalker) MapElem(m, k, v reflect.Value) error {
	if t.Keys == nil {
		t.Keys = make(map[string]bool)
		t.Values = make(map[interface{}]bool)
	}

	t.Keys[k.Interface().(string)] = true
	t.Values[v.Interface()] = true
	return nil
}

func TestWalk_MapWithPointers(t *testing.T) {
	w := new(TestInterfaceMapWalker)

	type S struct {
		Foo map[string]interface{}
	}

	a := "a"
	b := "b"

	data := &S{
		Foo: map[string]interface{}{
			"foo": &a,
			"bar": &b,
			"baz": 11,
			"zab": (*int)(nil),
		},
	}

	err := Walk(data, w)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(w.MapVal.Interface(), data.Foo) {
		t.Fatalf("Bad: %#v", w.MapVal.Interface())
	}

	expectedK := map[string]bool{"foo": true, "bar": true, "baz": true, "zab": true}
	if !reflect.DeepEqual(w.Keys, expectedK) {
		t.Fatalf("Bad keys: %#v", w.Keys)
	}

	expectedV := map[interface{}]bool{&a: true, &b: true, 11: true, (*int)(nil): true}
	if !reflect.DeepEqual(w.Values, expectedV) {
		t.Fatalf("Bad values: %#v", w.Values)
	}
}

type TestStructWalkerFieldSkip struct {
	Skip   bool
	Fields int
}

func (t *TestStructWalkerFieldSkip) Enter(l Location) error {
	if l == StructField {
		t.Fields++
	}

	return nil
}

func (t *TestStructWalkerFieldSkip) Exit(Location) error {
	return nil
}

func (t *TestStructWalkerFieldSkip) Struct(v reflect.Value) error {
	return nil
}

func (t *TestStructWalkerFieldSkip) StructField(sf reflect.StructField, v reflect.Value) error {
	if t.Skip && sf.Name[0] == '_' {
		return ErrSkipEntry
	}

	return nil
}

func TestWalkStructWithSkipEntry(t *testing.T) {
	data := &struct {
		Foo, _Bar int
	}{
		Foo:  1,
		_Bar: 2,
	}

	{
		var s TestStructWalkerFieldSkip
		if err := Walk(data, &s); err != nil {
			t.Fatalf("err: %s", err)
		}

		if s.Fields != 2 {
			t.Fatalf("bad: %d", s.Fields)
		}
	}

	{
		var s TestStructWalkerFieldSkip
		s.Skip = true
		if err := Walk(data, &s); err != nil {
			t.Fatalf("err: %s", err)
		}

		if s.Fields != 1 {
			t.Fatalf("bad: %d", s.Fields)
		}
	}
}

type TestStructWalkerValueSkip struct {
	Skip   bool
	Fields int
}

func (t *TestStructWalkerValueSkip) Enter(l Location) error {
	if l == StructField {
		t.Fields++
	}

	return nil
}

func (t *TestStructWalkerValueSkip) Exit(Location) error {
	return nil
}

func (t *TestStructWalkerValueSkip) Struct(v reflect.Value) error {
	if t.Skip {
		return ErrSkipEntry
	}

	return nil
}

func (t *TestStructWalkerValueSkip) StructField(sf reflect.StructField, v reflect.Value) error {
	return nil
}

func TestWalk_StructParentWithSkipEntry(t *testing.T) {
	data := &struct {
		Foo, _Bar int
	}{
		Foo:  1,
		_Bar: 2,
	}

	{
		var s TestStructWalkerValueSkip
		if err := Walk(data, &s); err != nil {
			t.Fatalf("err: %s", err)
		}

		if s.Fields != 2 {
			t.Fatalf("bad: %d", s.Fields)
		}
	}

	{
		var s TestStructWalkerValueSkip
		s.Skip = true
		if err := Walk(data, &s); err != nil {
			t.Fatalf("err: %s", err)
		}

		if s.Fields != 0 {
			t.Fatalf("bad: %d", s.Fields)
		}
	}
}
