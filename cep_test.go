package correios_test

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

import (
	"context"
	"testing"

	"github.com/gabstv/correios"
	"github.com/stretchr/testify/assert"
)

func TestConsultaCEP(t *testing.T) {
	// not found
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	r, err := correios.ConsultaCEP(ctx, "123456789")
	assert.Error(t, err)
	assert.Nil(t, r)
	// found
	r, err = correios.ConsultaCEP(ctx, "13056535")
	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, "13056535", r.CEP)
}
