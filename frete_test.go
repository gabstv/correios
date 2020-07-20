package correios

import (
	"testing"

	"golang.org/x/net/context"
)

func TestSimpleRequest(t *testing.T) {
	r := NewFreteRequest("01243000", "65299970").SetServicos(SvcSEDEXVarejo)
	resp, err := CalcularFrete(context.Background(), r)
	if err != nil {
		t.Fail()
	}
	_ = resp
}
