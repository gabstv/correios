![Package](package.png)

CORREIOS
========

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/gabstv/correios"
)

func main() {
    // cria um FreteRequest com os defaults:
    // PesoKg         0.5
    // ComprimentoCm 16.0
    // LarguraCm     11.0
    // AlturaCm       5.0
    // Servicos      []{SvcSEDEXVarejo, SvcPACVarejo}
    r := correios.NewFreteRequest("01243000", "65299970")
    resp, err := correios.CalcularFrete(context.Background(), r)

    if err != nil {
        fmt.Println("failed:", err.Error())
        os.Exit(1)
    }
    for svctype, svc := range resp.Servicos {
        fmt.Println(svctype.String(), "R$", svc.Preco.String(), " - prazo:", svc.PrazoEntregaDias)
    }
}
```