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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var (
	ConsultaCEPURL       = "https://buscacepinter.correios.com.br/app/endereco/carrega-cep-endereco.php"
	ConsultaCEPReferer   = "https://buscacepinter.correios.com.br/app/endereco/index.php"
	ConsultaCEPUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1 Safari/605.1.15"
	ErrNoResults         = errors.New("correios: no results")
)

// CEPResult is the result of a ConsultaCEP request.
type CEPResult struct {
	CEP        string `json:"cep"`
	UF         string `json:"uf"`
	Cidade     string `json:"cidade"`
	Bairro     string `json:"bairro"`
	Logradouro string `json:"logradouro"`
}

// RawCEPResult is the raw data of a ConsultaCEP request.
type RawCEPResult struct {
	Erro     bool   `json:"erro"`
	Mensagem string `json:"mensagem"`
	Total    int    `json:"total"`
	Dados    []struct {
		Uf                       string        `json:"uf"`
		Localidade               string        `json:"localidade"`
		LocNoSem                 string        `json:"locNoSem"`
		LocNu                    string        `json:"locNu"`
		LocalidadeSubordinada    string        `json:"localidadeSubordinada"`
		LogradouroDNEC           string        `json:"logradouroDNEC"`
		LogradouroTextoAdicional string        `json:"logradouroTextoAdicional"`
		LogradouroTexto          string        `json:"logradouroTexto"`
		Bairro                   string        `json:"bairro"`
		BaiNu                    string        `json:"baiNu"`
		NomeUnidade              string        `json:"nomeUnidade"`
		Cep                      string        `json:"cep"`
		TipoCep                  string        `json:"tipoCep"`
		NumeroLocalidade         string        `json:"numeroLocalidade"`
		Situacao                 string        `json:"situacao"`
		FaixasCaixaPostal        []interface{} `json:"faixasCaixaPostal"`
		FaixasCep                []interface{} `json:"faixasCep"`
	} `json:"dados"`
}

// ConsultaCEP returns the street, city, UF and district (bairro) of a brazillian ZIP code.
func ConsultaCEP(ctx context.Context, cep string) (*CEPResult, error) {
	vals := url.Values{}
	vals.Set("MIME Type", "application/x-www-form-urlencoded; charset=utf-8")
	vals.Set("pagina", "/app/endereco/index.php")
	vals.Set("cepaux", "")
	vals.Set("mensagem_alerta", "")
	vals.Set("endereco", FilterCEP(cep))
	vals.Set("tipoCEP", "ALL")
	buf := bytes.NewBufferString(vals.Encode())
	rq0, err := http.NewRequestWithContext(ctx, http.MethodPost, ConsultaCEPURL, buf)
	if err != nil {
		return nil, err
	}
	rq0.Header.Set("Referer", ConsultaCEPReferer)
	rq0.Header.Set("User-Agent", ConsultaCEPUserAgent)
	rq0.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	cresp, err := http.DefaultClient.Do(rq0)
	if err != nil {
		return nil, err
	}
	defer cresp.Body.Close()
	if cresp.StatusCode != http.StatusOK {
		return nil, errors.New("http status: " + cresp.Status)
	}
	rawResp := &RawCEPResult{}
	if err := json.NewDecoder(cresp.Body).Decode(rawResp); err != nil {
		return nil, fmt.Errorf("decode json error: %w", err)
	}
	if rawResp.Erro {
		return nil, errors.New("correios: " + rawResp.Mensagem)
	}
	if rawResp.Total == 0 || len(rawResp.Dados) == 0 {
		return nil, ErrNoResults
	}
	result := &CEPResult{
		CEP:    rawResp.Dados[0].Cep,
		UF:     rawResp.Dados[0].Uf,
		Cidade: rawResp.Dados[0].Localidade,
		Bairro: rawResp.Dados[0].Bairro,
	}
	if rawResp.Dados[0].LogradouroDNEC != "" {
		result.Logradouro = rawResp.Dados[0].LogradouroDNEC
	} else if rawResp.Dados[0].LogradouroTexto != "" {
		result.Logradouro = rawResp.Dados[0].LogradouroTexto
	}
	return result, nil
}
