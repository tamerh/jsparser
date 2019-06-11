package jsparser

import (
	"bufio"
	"flag"
	"os"
	"strings"
	"testing"
)

var minify bool

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags

	flag.BoolVar(&minify, "minify", false, "Minify")

	flag.Parse()

	os.Exit(m.Run())
}

func getparser(prop string) *JsonParser {

	if minify {
		// todo add some space after some values
		const minijson string = `{"nu":null,"b":true,"b1":false,"n":2323,"n1":23.23,"n2":23.23e-6 ,"s":"sstring","s1":"s1tring","s2":"s2tr\\ing\"蒜","o":{"o1":"o1string","o2":"o2string","o3":true,"o4":["o4string",{"o41":"o41string"},["o4nestedarray item 1","o4nestedarray item 1 item 2",true,99,null,90.98]],"o5":98.21,"o6":null,"o7":{"o71":"o71string","o72":["o72string",null,false,98,{}],"o73":true,"o74":98}},"a":[{"a11":"o71string\\","a12":["o72string",null,false,98,{}],"a13":true,"a14":98},{"a11":"o71string","a12":["o72string",null,false,98,{}],"a13":true,"a14":98},"astringinside",false,99,null,0.00043333]}`

		br := bufio.NewReader(strings.NewReader(minijson))

		p := NewJsonParser(br, prop)

		return p
	}

	file, _ := os.Open("sample.json")

	br := bufio.NewReader(file)

	p := NewJsonParser(br, prop)

	return p

}

func TestString(t *testing.T) {

	var js JSON

	p := getparser("s")
	resultCount := 0

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++

	}

	if resultCount != 1 {
		panic("result count must 1")
	}

	if js.StringVal != "sstring" {
		panic("invalid result string")
	}

	if js.ValueType != String {
		panic("Value type must be string")
	}

	p = getparser("s2")

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json

	}

	if js.StringVal != "s2tr\\ing\"蒜" {
		panic("invalid result string")
	}

	// Skip

}

func TestBoolean(t *testing.T) {

	p := getparser("b")

	resultCount := 0
	var js JSON

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++

	}

	if resultCount != 1 {
		panic("result count must 1")
	}

	if !js.BoolVal {
		panic("invalid result boolean")
	}

	if js.ValueType != Boolean {
		panic("Value type must be boolean")
	}

}

func TestNumber(t *testing.T) {

	p := getparser("n2")

	resultCount := 0
	var js JSON

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++

	}

	if resultCount != 1 {
		panic("result count must 1")
	}

	if js.StringVal != "23.23e-6" {
		panic("invalid result")
	}

	if js.ValueType != Number {
		panic("Value type must be boolean")
	}

}

func TestNull(t *testing.T) {

	p := getparser("nu")

	resultCount := 0
	var js JSON

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++

	}

	if resultCount != 1 {
		panic("result count must 1")
	}

	if js.StringVal != "" {
		panic("invalid result")
	}

	if js.ValueType != Null {
		panic("Value type must be null")
	}

}

func TestObject(t *testing.T) {

	p := getparser("o")

	resultCount := 0
	var js JSON

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++

	}

	if resultCount != 1 {
		panic("result count must 1")
	}

	if js.ValueType != Object {
		panic("Value type must be object")
	}

	if val, ok := js.ObjectVals["o1"]; !ok || val.StringVal != "o1string" || val.ValueType != String {
		panic("Test failed")
	}

	if val, ok := js.ObjectVals["o2"]; !ok || val.StringVal != "o2string" || val.ValueType != String {
		panic("Test failed")
	}

	if val, ok := js.ObjectVals["o3"]; !ok || !val.BoolVal || val.ValueType != Boolean {
		panic("Test failed")
	}

	if val, ok := js.ObjectVals["o4"]; !ok || val.ValueType != Array || len(val.ArrayVals) != 3 {
		panic("Test failed")
	}

	if val := js.ObjectVals["o4"]; val.ValueType != Array || len(val.ArrayVals[2].ArrayVals) != 6 {
		panic("Test failed")
	}

	// Skip test
	p = getparser("o").SkipProps([]string{"o1", "o2", "o4", "o5", "o6", "o7"})

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		js = *json
		resultCount++
	}

	if _, ok := js.ObjectVals["o1"]; ok {
		panic("Test failed")
	}

	if _, ok := js.ObjectVals["o2"]; ok {
		panic("Test failed")
	}

	if _, ok := js.ObjectVals["o4"]; ok {
		panic("Test failed")
	}

	if _, ok := js.ObjectVals["o5"]; ok {
		panic("Test failed")
	}

	if _, ok := js.ObjectVals["o6"]; ok {
		panic("Test failed")
	}

	if _, ok := js.ObjectVals["o7"]; ok {
		panic("Test failed")
	}

	if val, ok := js.ObjectVals["o3"]; !ok || !val.BoolVal || val.ValueType != Boolean {
		panic("Test failed")
	}

}

func TestArray(t *testing.T) {

	p := getparser("a")

	var results []*JSON

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}
		results = append(results, json)
	}

	if len(results) != 7 {
		panic("result count must 7")
	}

	if results[0].ValueType != Object {
		panic("Value type must be object")
	}
	if results[1].ValueType != Object {
		panic("Value type must be object")
	}

	if results[2].ValueType != String {
		panic("Value type must be string")
	}

	if results[3].ValueType != Boolean {
		panic("Value type must be bool")
	}

	if results[4].ValueType != Number {
		panic("Value type must be bool")
	}

	if results[5].ValueType != Null {
		panic("Value type must be null")
	}

	if results[6].ValueType != Number {
		panic("Value type must be bool")
	}

	// Skip test
	p = getparser("a").SkipProps([]string{"a11", "a12", "a13"})

	for json := range *p.Stream() {

		if json.Err != nil {
			panic(json.Err)
		}

		if json.ValueType == Object {

			if _, ok := json.ObjectVals["a11"]; ok {
				panic("Test failed")
			}

			if _, ok := json.ObjectVals["a12"]; ok {
				panic("Test failed")
			}

			if _, ok := json.ObjectVals["a13"]; ok {
				panic("Test failed")
			}

		}

	}

}
