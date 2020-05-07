package jsparser

import "unicode/utf8"

// based on https://github.com/bcicen/jstream

type scratch struct {
	data    []byte
	dataRes []*JSON
	fill    int
	fillRes int
}

// reset scratch buffer
func (s *scratch) reset() { s.fill = 0 }

// bytes returns the written contents of scratch buffer
func (s *scratch) bytes() []byte { return s.data[0:s.fill] }

// string returns the written contents of scratch buffer
func (s *scratch) string() string { return string(s.data[0:s.fill]) }

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

// grow result buffer
func (s *scratch) growRes() {
	ndata := make([]*JSON, cap(s.dataRes)*2)
	copy(ndata, s.dataRes[:])
	s.dataRes = ndata
}

// add result
func (s *scratch) addRes(res *JSON) {
	if s.fillRes+1 >= cap(s.dataRes) {
		s.growRes()
	}

	s.dataRes[s.fillRes] = res
	s.fillRes++
}

func (s *scratch) allRes() []*JSON {
	return s.dataRes[0:s.fillRes]
}
