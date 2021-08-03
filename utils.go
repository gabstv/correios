// Copyright (c) 2021 Gabriel Ochsenhofer

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package correios

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// FilterCEP removes non numbers from a CEP.
func FilterCEP(v string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, v)
}

func fixWrongDecimals(ds string) string {
	return strings.Replace(ds, ",", ".", -1)
}

type CharsetISO88591er struct {
	r   io.ByteReader
	buf *bytes.Buffer
}

func NewCharsetISO88591(r io.Reader) *CharsetISO88591er {
	buf := bytes.NewBuffer(make([]byte, 0, utf8.UTFMax))
	return &CharsetISO88591er{r.(io.ByteReader), buf}
}

func (cs *CharsetISO88591er) ReadByte() (b byte, err error) {
	// http://unicode.org/Public/MAPPINGS/ISO8859/8859-1.TXT
	// Date: 1999 July 27; Last modified: 27-Feb-2001 05:08
	if cs.buf.Len() <= 0 {
		r, err := cs.r.ReadByte()
		if err != nil {
			return 0, err
		}
		if r < utf8.RuneSelf {
			return r, nil
		}
		cs.buf.WriteRune(rune(r))
	}
	return cs.buf.ReadByte()
}

func (cs *CharsetISO88591er) Read(p []byte) (int, error) {
	// Use ReadByte method.
	return 0, os.ErrInvalid
}

func isCharset(charset string, names []string) bool {
	charset = strings.ToLower(charset)
	for _, n := range names {
		if charset == strings.ToLower(n) {
			return true
		}
	}
	return false
}

func IsCharsetISO88591(charset string) bool {
	// http://www.iana.org/assignments/character-sets
	// (last updated 2010-11-04)
	names := []string{
		// Name
		"ISO_8859-1:1987",
		// Alias (preferred MIME name)
		"ISO-8859-1",
		// Aliases
		"iso-ir-100",
		"ISO_8859-1",
		"latin1",
		"l1",
		"IBM819",
		"CP819",
		"csISOLatin1",
	}
	return isCharset(charset, names)
}

func IsCharsetUTF8(charset string) bool {
	names := []string{
		"UTF-8",
		// Default
		"",
	}
	return isCharset(charset, names)
}

func CharsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch {
	case IsCharsetUTF8(charset):
		return input, nil
	case IsCharsetISO88591(charset):
		return NewCharsetISO88591(input), nil
	}
	return nil, errors.New("CharsetReader: unexpected charset: " + charset)
}
