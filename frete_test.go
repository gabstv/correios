package correios

import (
	"testing"

	"golang.org/x/net/context"
)

func TestSimpleRequest(t *testing.T) {
	StupidSingleServiceMode = true
	r := NewFreteRequest("01243000", "65299970")
	resp, err := CalcularFrete(context.Background(), r)
	if err != nil {
		t.Fail()
	}
	_ = resp
}
