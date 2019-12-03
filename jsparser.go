package jsparser

import (
	"bufio"
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

type JsonParser struct {
	reader        *bufio.Reader
	loopProp      string
	resultChannel chan *JSON
	skipProps     map[string]bool
	TotalReadSize uint64
	lastReadSize  int
	scratch       *scratch
}

// JSON parsed result
type JSON struct {
	StringVal  string
	BoolVal    bool
	ArrayVals  []*JSON
	ObjectVals map[string]*JSON
	ValueType  ValueType
	Err        error
}

// ValueType of JSON value
type ValueType int

// JSON types
const (
	Invalid ValueType = iota
	Null
	String
	Number
	Boolean
	Array
	Object
)

func NewJSONParser(reader *bufio.Reader, loopProp string) *JsonParser {

	j := &JsonParser{
		reader:        reader,
		loopProp:      loopProp,
		resultChannel: make(chan *JSON, 256),
		skipProps:     map[string]bool{},
		scratch:       &scratch{data: make([]byte, 1024)},
	}
	return j
}

func (j *JsonParser) SkipProps(skipProps []string) *JsonParser {

	if len(skipProps) > 0 {
		for _, s := range skipProps {
			j.skipProps[s] = true
		}
	}
	return j

}

func (j *JsonParser) Stream() chan *JSON {

	go j.parse()

	return j.resultChannel

}

func (j *JsonParser) parse() {

	defer close(j.resultChannel)

	var b byte
	var err error
	var res *JSON

	for {
		b, err = j.readByte()

		if err != nil {
			return
		}

		if j.isWS(b) {
			continue
		}

		if b == '"' { // begining of possible json property

			prop, isprop, propErr := j.getPropName()

			if propErr {
				j.sendError()
				return
			}

			if isprop {

				b, err = j.skipWS()
				if err != nil {
					j.sendError()
					return
				}

				valType, typeErr := j.getValueType(b)

				if typeErr != nil {
					j.sendError()
					return
				}

				if j.loopProp == prop {

					switch valType {
					case String:

						res = j.string()
						j.resultChannel <- res

						if res.Err != nil {
							return
						}

					case Array:

						success := j.loopArray()
						if !success {
							return
						}

					case Object:

						res = &JSON{ObjectVals: map[string]*JSON{}, ValueType: Object}
						j.getObjectValueTree(res)
						j.resultChannel <- res

						if res.Err != nil {
							return
						}

					case Boolean:

						res = j.boolean()
						j.resultChannel <- res

						if res.Err != nil {
							return
						}

					case Number:

						res = j.number(b)
						j.resultChannel <- res

						if res.Err != nil {
							return
						}

					case Null:

						res = j.null()
						j.resultChannel <- res

						if res.Err != nil {
							return
						}

					}

				} else {

					if valType == String { // if valtype is string just skip it otherwise continue looking loopProp.
						res = j.skipString()
						if res.Err != nil {
							j.resultChannel <- res
							return
						}
					}

				}
			}
		}
	}

}

func (j *JsonParser) loopArray() bool {

	var b byte
	var err error
	var res *JSON

	for {

		b, err = j.skipWS()

		if err != nil {
			j.sendError()
			return false
		}

		if b == ']' {
			return true
		}

		if b == ',' {
			continue
		}

		valType, err := j.getValueType(b)

		if err != nil {
			j.sendError()
			return false
		}

		switch valType {
		case String:

			res = j.string()
			j.resultChannel <- res

		case Array:

			res = &JSON{ObjectVals: map[string]*JSON{}, ValueType: Array}
			j.getArrayValueTree(res)
			j.resultChannel <- res

		case Object:

			res = &JSON{ObjectVals: map[string]*JSON{}, ValueType: Object}
			j.getObjectValueTree(res)
			j.resultChannel <- res

		case Boolean:

			res = j.boolean()
			j.resultChannel <- res

		case Number:

			res = j.number(b)
			j.resultChannel <- res

		case Null:

			res = j.null()
			j.resultChannel <- res

		}

		if res.Err != nil {
			return false
		}

	}

}

func (j *JsonParser) getObjectValueTree(result *JSON) *JSON {

	if result.Err != nil {
		return result
	}

	var b byte
	var err error
	var res *JSON
	for {

		b, err = j.readByte()

		if err != nil {
			return j.resultError()
		}

		if j.isWS(b) {
			continue
		}

		if b == '"' { // begining of json property

			prop, isprop, err2 := j.getPropName()

			if err2 {
				result.Err = j.defaultError()
				return result
			}

			if !isprop { // look again
				continue
			}

			b, err = j.skipWS()
			if err != nil {
				result.Err = j.defaultError()
				return result
			}

			valType, err := j.getValueType(b)

			if err != nil {
				result.Err = err
				return result
			}

			switch valType {
			case String:

				if ok := j.skipProps[prop]; ok {
					res = j.skipString()
					break
				}

				res = j.string()
				result.ObjectVals[prop] = res

			case Array:

				if ok := j.skipProps[prop]; ok {
					res = j.skipArrayOrObject('[', ']')
					break
				}

				res = j.getArrayValueTree(&JSON{ValueType: Array})
				result.ObjectVals[prop] = res

			case Object:

				if ok := j.skipProps[prop]; ok {
					res = j.skipArrayOrObject('{', '}')
					break
				}

				res = j.getObjectValueTree(&JSON{ObjectVals: map[string]*JSON{}, ValueType: Object})
				result.ObjectVals[prop] = res

			case Boolean:

				res = j.boolean()
				// rest of the skip since they are small we just don't include in the result
				if ok := j.skipProps[prop]; !ok {
					result.ObjectVals[prop] = res
				}

			case Number:

				res = j.number(b)
				if ok := j.skipProps[prop]; !ok {
					result.ObjectVals[prop] = res
				}

			case Null:

				res = j.null()
				if ok := j.skipProps[prop]; !ok {
					result.ObjectVals[prop] = res
				}

			}

			if res.Err != nil {
				result.Err = res.Err
				return result
			}

		} else if b == ',' {

			continue

		} else if b == '}' { // completion of current object

			return result

		} else { // invalid end

			result.Err = j.defaultError()
			return result

		}

	}

}

func (j *JsonParser) getArrayValueTree(result *JSON) *JSON {

	if result.Err != nil {
		return result
	}

	var b byte
	var err error
	var res *JSON

	for {

		b, err = j.readByte()

		if err != nil {
			return j.resultError()
		}

		if j.isWS(b) {
			continue
		}

		if b == ',' {
			continue
		}

		if b == ']' { // means complete of current array
			return result
		}

		valType, err := j.getValueType(b)

		if err != nil {
			return j.resultError()
		}
		switch valType {
		case String:

			res = j.string()
			result.ArrayVals = append(result.ArrayVals, res)

		case Array:

			res = j.getArrayValueTree(&JSON{ValueType: Array})
			result.ArrayVals = append(result.ArrayVals, res)

		case Object:

			res = j.getObjectValueTree(&JSON{ObjectVals: map[string]*JSON{}, ValueType: Object})
			result.ArrayVals = append(result.ArrayVals, res)

		case Boolean:

			res = j.boolean()
			result.ArrayVals = append(result.ArrayVals, res)

		case Number:

			res = j.number(b)
			result.ArrayVals = append(result.ArrayVals, res)

		case Null:

			res = j.null()
			result.ArrayVals = append(result.ArrayVals, res)

		}

		if res.Err != nil {
			result.Err = res.Err
			return result
		}

	}

}

func (j *JsonParser) number(first byte) *JSON {

	var c byte
	var err error
	j.scratch.reset()
	j.scratch.add(first)

	for {

		c, err = j.readByte()

		if err != nil {
			return &JSON{Err: j.defaultError(), ValueType: Invalid}
		}

		if j.isWS(c) {

			c, err = j.skipWS()

			if err != nil {
				return j.resultError()
			}

			if !(c == ',' || c == '}' || c == ']') {
				return j.resultError()
			}
			err := j.unreadByte()
			if err != nil {
				return j.resultError()
			}

			return &JSON{StringVal: string(j.scratch.bytes()), ValueType: Number}
		}

		if c == ',' || c == '}' || c == ']' {

			err := j.unreadByte()
			if err != nil {
				return j.resultError()
			}

			return &JSON{StringVal: string(j.scratch.bytes()), ValueType: Number}
		}

		j.scratch.add(c)

	}

}

func (j *JsonParser) boolean() *JSON {

	var c byte
	var err error

	c, err = j.readByte()

	if err != nil {
		return j.resultError()
	}

	// true
	if c == 'r' {
		c, err = j.readByte()

		if err != nil {
			return j.resultError()
		}
		if c == 'u' {
			c, err = j.readByte()

			if err != nil {
				return j.resultError()
			}
			if c == 'e' {
				// check last
				c, err = j.skipWS()
				if err != nil {
					return j.resultError()
				}
				if !(c == ',' || c == '}' || c == ']') {
					return j.resultError()
				}
				err := j.unreadByte()
				if err != nil {
					return j.resultError()
				}

				return &JSON{BoolVal: true, ValueType: Boolean}
			}
		}
	}

	// false
	if c == 'a' {
		c, err = j.readByte()

		if err != nil {
			return j.resultError()
		}
		if c == 'l' {
			c, err = j.readByte()

			if err != nil {
				return j.resultError()
			}
			if c == 's' {
				c, err = j.readByte()

				if err != nil {
					return j.resultError()
				}
				if c == 'e' {
					// check last
					c, err = j.skipWS()
					if err != nil {
						return j.resultError()
					}
					if !(c == ',' || c == '}' || c == ']') {
						return j.resultError()
					}
					err := j.unreadByte()
					if err != nil {
						return j.resultError()
					}

					return &JSON{BoolVal: false, ValueType: Boolean}
				}
			}
		}
	}

	return j.resultError()

}

func (j *JsonParser) null() *JSON {

	var c byte
	var err error

	c, err = j.readByte()

	if err != nil {
		return j.resultError()
	}

	// true
	if c == 'u' {
		c, err = j.readByte()

		if err != nil {
			return j.resultError()
		}

		if c == 'l' {
			c, err = j.readByte()

			if err != nil {
				return j.resultError()
			}
			if c == 'l' {
				// check last
				c, err = j.skipWS()
				if err != nil {
					return j.resultError()
				}

				if !(c == ',' || c == '}' || c == ']') {
					return j.resultError()
				}

				err := j.unreadByte()
				if err != nil {
					return j.resultError()
				}

				return &JSON{ValueType: Null}
			}
		}
	}

	return j.resultError()
}

func (j *JsonParser) skipString() *JSON {

	var c byte
	var prev byte
	var prevPrev byte
	var err error
	for {

		c, err = j.readByte()

		if err != nil {
			return j.resultError()
		}

		if c == '"' {

			if !(prev == '\\' && prevPrev != '\\') { // escape check
				return &JSON{}
			}

		}

		prevPrev = prev
		prev = c

	}

}

func (j *JsonParser) skipArrayOrObject(start byte, end byte) *JSON {

	var c byte
	var err error
	var depth = 1
	for {

		c, err = j.readByte()

		if err != nil {
			return j.resultError()
		}

		switch c {
		case '"':
			res := j.skipString() // this is needed because string can contain [ or ]
			if res.Err != nil {
				return res
			}
		case start:
			depth++
		case end:
			depth--
			if depth == 0 {
				return &JSON{}
			}

		}

	}

}

func (j *JsonParser) getValueType(c byte) (ValueType, error) {

	switch c {
	case '"':
		return String, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
		return Number, nil
	case 'f':
		return Boolean, nil
	case 't':
		return Boolean, nil
	case 'n':
		return Null, nil
	case '[':
		return Array, nil
	case '{':
		return Object, nil
	}

	return Invalid, j.defaultError()

}

func (j *JsonParser) getPropName() (string, bool, bool) {

	res := j.string()

	if res.Err != nil {
		return "", false, true
	}

	b, err := j.skipWS()

	if err != nil {
		return "", false, true
	}

	if b == ':' { // end of property name
		return res.StringVal, true, false
	}

	err = j.unreadByte()

	if err != nil {
		return "", false, true
	}

	return res.StringVal, false, false

}

func (j *JsonParser) isWS(in byte) bool {

	if in == ' ' || in == '\n' || in == '\t' || in == '\r' {
		return true
	}

	return false

}

// skips WS and read first non WS
func (j *JsonParser) skipWS() (byte, error) {

	var b byte
	var err error
	for {
		b, err = j.readByte()
		if err != nil {
			return 0, err
		}
		if b == ' ' || b == '\n' || b == '\t' || b == '\r' {
			continue
		} else {
			return b, nil
		}
	}

}

func (j *JsonParser) readByte() (byte, error) {

	by, err := j.reader.ReadByte()

	j.TotalReadSize = j.TotalReadSize + 1

	j.lastReadSize = 1

	if err != nil {
		return 0, err
	}
	return by, nil

}

func (j *JsonParser) unreadByte() error {

	err := j.reader.UnreadByte()
	if err != nil {
		return err
	}
	j.TotalReadSize = j.TotalReadSize - 1
	return nil

}

func (j *JsonParser) sendError() {
	err := fmt.Errorf("Invalid json")
	j.resultChannel <- &JSON{Err: err, ValueType: Invalid}
}

func (j *JsonParser) resultError() *JSON {

	return &JSON{Err: j.defaultError(), ValueType: Invalid}

}

func (j *JsonParser) defaultError() error {
	err := fmt.Errorf("Invalid json")
	return err
}

// rest of the part taken and adapted from jstream
//https://github.com/bcicen/jstream
// string called by `any` or `object`(for map keys) after reading `"`
func (j *JsonParser) string() *JSON {

	j.scratch.reset()

	var err error
	var c byte

	c, err = j.readByte()
	if err != nil {
		if err != nil {
			return j.resultError()
		}
	}

scan:
	for {
		switch {
		case c == '"':
			return &JSON{StringVal: string(j.scratch.bytes()), ValueType: String}
		case c == '\\':
			c, err = j.readByte()
			if err != nil {
				if err != nil {
					return j.resultError()
				}
			}
			goto scan_esc
		case c < 0x20:
			err := fmt.Errorf("Invalid json")
			return &JSON{Err: err, ValueType: Invalid}
			// Coerce to well-formed UTF-8.

		}
		j.scratch.add(c)
		c, err = j.readByte()
		if err != nil {
			if err != nil {
				return j.resultError()
			}
		}
	}

scan_esc:
	switch c {
	case '"', '\\', '/', '\'':
		j.scratch.add(c)
	case 'u':
		goto scan_u
	case 'b':
		j.scratch.add('\b')
	case 'f':
		j.scratch.add('\f')
	case 'n':
		j.scratch.add('\n')
	case 'r':
		j.scratch.add('\r')
	case 't':
		j.scratch.add('\t')
	default:
		//err := fmt.Errorf("error in string escape code")
		return j.resultError()
	}

	c, err = j.readByte()
	if err != nil {
		if err != nil {
			return j.resultError()
		}
	}

	goto scan

scan_u:
	r := j.u4()
	if r < 0 {
		//err := fmt.Errorf("in unicode escape sequence")
		return j.resultError()
	}

	// check for proceeding surrogate pair
	c, err = j.readByte()
	if err != nil {
		if err != nil {
			return j.resultError()
		}
	}

	if !utf16.IsSurrogate(r) || c != '\\' {
		j.scratch.addRune(r)
		goto scan
	}

	c, err = j.readByte()
	if err != nil {
		if err != nil {
			return j.resultError()
		}
	}

	if c != 'u' {
		j.scratch.addRune(r)
		goto scan_esc
	}

	r2 := j.u4()
	if r2 < 0 {
		return j.resultError()
	}

	// write surrogate pair
	j.scratch.addRune(utf16.DecodeRune(r, r2))

	c, err = j.readByte()
	if err != nil {
		if err != nil {
			return j.resultError()
		}
	}

	goto scan
}

// u4 reads four bytes following a \u escape
func (j *JsonParser) u4() rune {
	// logic taken from:
	// github.com/buger/jsonparser/blob/master/escape.go#L20

	var c byte
	var err error
	var h [4]int
	for i := 0; i < 4; i++ {

		c, err = j.readByte()
		if err != nil {
			if err != nil {
				return -1
			}
		}
		switch {
		case c >= '0' && c <= '9':
			h[i] = int(c - '0')
		case c >= 'A' && c <= 'F':
			h[i] = int(c - 'A' + 10)
		case c >= 'a' && c <= 'f':
			h[i] = int(c - 'a' + 10)
		default:
			return -1
		}
	}
	return rune(h[0]<<12 + h[1]<<8 + h[2]<<4 + h[3])
}

type scratch struct {
	data []byte
	fill int
}

// reset scratch buffer
func (s *scratch) reset() { s.fill = 0 }

// bytes returns the written contents of scratch buffer
func (s *scratch) bytes() []byte { return s.data[0:s.fill] }

// grow scratch buffer
func (s *scratch) grow() {
	ndata := make([]byte, cap(s.data)*2)
	copy(ndata, s.data[:])
	s.data = ndata
}

// append single byte to scratch buffer
func (s *scratch) add(c byte) {
	if s.fill+1 >= cap(s.data) {
		s.grow()
	}

	s.data[s.fill] = c
	s.fill++
}

// append encoded rune to scratch buffer
func (s *scratch) addRune(r rune) int {
	if s.fill+utf8.UTFMax >= cap(s.data) {
		s.grow()
	}

	n := utf8.EncodeRune(s.data[s.fill:], r)
	s.fill += n
	return n
}
