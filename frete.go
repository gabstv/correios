package correios

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

// RequestMode altera o metodo utilizado p/ obter o frete
type RequestMode int

const (
	// RequestModeAuto escolher o modo automaticamente
	RequestModeAuto RequestMode = iota
	// RequestModeSingle sempre envia um request por tipo de serviço
	RequestModeSingle
	// RequestModeCombined sempre envia um único request com todos tipos de serviço
	RequestModeCombined
)

// FreteEndpoint o endpoint a ser utilizado para calcular o frete
var FreteEndpoint = "http://ws.correios.com.br/calculador/CalcPrecoPrazo.aspx"

// TipoServico representa os tipos de serviço (numérico)
type TipoServico string

// TipoErro representa uma resposta de erro dos Correios (código interno)
type TipoErro int

// update 05/05/2017
// Correios alterou os códigos de serviço:
// 04510 – PAC sem contrato (era 41106)
// 04669 – PAC com contrato
// 04014 – SEDEX sem contrato (era 40010)
// 04162 – SEDEX com contrato
//
// Não há nenhuma documentação sobre a mudança no momento.

// Todos os tipos de serviço
//https://www.correios.com.br/para-voce/correios-de-a-a-z/pdf/calculador-remoto-de-precos-e-prazos/manual-de-implementacao-do-calculo-remoto-de-precos-e-prazos
const (
	SvcSEDEXVarejo        TipoServico = "04014"
	SvcSEDEXACobrarVarejo TipoServico = "40045"
	SvcSEDEX10Varejo      TipoServico = "40215"
	SvcSEDEXHojeVarejo    TipoServico = "40290"
	SvcPACVarejo          TipoServico = "04510"
	SvcPACComContrato     TipoServico = "04669"
	SvcSEDEXComContrato   TipoServico = "04162"
)

// Todos os tipos de erros possíveis que a API dos Correios pode retornar
const (
	ErrTipoServicoInvalido          TipoErro = -1
	ErrCepOrigemInvalido            TipoErro = -2
	ErrCepDestinoInvalido           TipoErro = -3
	ErrCepPesoExcedido              TipoErro = -4
	ErrValorDeclaradoAlto10k        TipoErro = -5
	ErrServicoIndisponivelTrecho    TipoErro = -6
	ErrValorDeclaradoObrigatorio    TipoErro = -7
	ErrMaoPropriaIndisponivel       TipoErro = -8
	ErrAvisoRecebimentoIndisponivel TipoErro = -9
	ErrPrecificacaoIndisponivel     TipoErro = -10
	ErrInformarDimensoes            TipoErro = -11
	ErrComprimento                  TipoErro = -12
	ErrLargura                      TipoErro = -13
	ErrAltura                       TipoErro = -14
	ErrComprimento105               TipoErro = -15  // > 105cm
	ErrLargura105                   TipoErro = -16  // > 105cm
	ErrAltura105                    TipoErro = -17  // > 105cm
	ErrAlturaInferior               TipoErro = -18  // < 2cm
	ErrLarguraInferior              TipoErro = -20  // < 11cm
	ErrComprimentoInferior          TipoErro = -22  // < 16cm
	ErrDimensoesSoma                TipoErro = -23  // A soma resultante do comprimento + largura + altura não deve superar a 200 cm
	ErrComprimento2                 TipoErro = -24  // WTF (ver -12)
	ErrDiametro                     TipoErro = -25  // Diâmetro inválido
	ErrComprimento3                 TipoErro = -26  // WTF (ver -12)
	ErrDiametro2                    TipoErro = -27  // ?
	ErrComprimento4                 TipoErro = -28  // O comprimento não pode ser maior que 105 cm.
	ErrDiametro91                   TipoErro = -29  // O diâmetro não pode ser maior que 91 cm.
	ErrComprimento18                TipoErro = -30  // O comprimento não pode ser inferior a 18 cm.
	ErrDiametro5                    TipoErro = -31  // O diâmetro não pode ser inferior a 5 cm.
	ErrSomaDiametro                 TipoErro = -32  // A soma resultante do comprimento + o dobro do diâmetro não deve superar a 200 cm
	ErrSistemaIndisponivel          TipoErro = -33  // Sistema temporariamente fora do ar. Favor tentar mais tarde.
	ErrCodigoOuSenha                TipoErro = -34  // Código Administrativo ou Senha inválidos.
	ErrSenha                        TipoErro = -35  // Senha incorreta.
	ErrSemContrato                  TipoErro = -36  // Cliente não possui contrato vigente com os Correios.
	ErrSemServicoAtivo              TipoErro = -37  // Cliente não possui serviço ativo em seu contrato.
	ErrServicoIndisponivelAdmin     TipoErro = -38  // Serviço indisponível para este código administrativo.
	ErrPesoExcedidoEnvelope         TipoErro = -39  // Peso excedido para o formato envelope
	ErrInformarDimensoes2           TipoErro = -40  // Para definicao do preco deverao ser informados, tambem, o comprimento e a largura e altura do objeto em centimetros (cm).
	ErrComprimento60                TipoErro = -41  // O comprimento nao pode ser maior que 60 cm.
	ErrComprimento16                TipoErro = -42  // (repetido) O comprimento nao pode ser inferior a 16 cm.
	ErrComprimentoLargura120        TipoErro = -43  // A soma resultante do comprimento + largura nao deve superar a 120 cm
	ErrLarguraInferior2             TipoErro = -44  // < 11cm
	ErrLarguraSuperior60            TipoErro = -44  // > 60cm
	ErrErroCalculoTarifa            TipoErro = -888 // Erro ao calcular a tarifa
	ErrLocalidadeOrigem             TipoErro = 006  // Localidade de origem não abrange o serviço informado
	ErrLocalidadeDestino            TipoErro = 007  // Localidade de destino não abrange o serviço informado
	ErrServicoIndisponivelTrecho2   TipoErro = 8    // 008 Serviço indisponível para o trecho informado
	ErrAreaDeRiscoCEPInicial        TipoErro = 9    // 009 CEP inicial pertencente a Área de Risco.
	ErrAreaPrazoDiferenciado        TipoErro = 010  // Área com entrega temporariamente sujeita a prazo diferenciado.
	ErrAreaDeRiscoCEPs              TipoErro = 011  // CEP inicial e final pertencentes a Área de Risco
	ErrIndisponivel                 TipoErro = 7    // Serviço indisponível, tente mais tarde
	ErrIndeterminado                TipoErro = 99   // Outros erros diversos do .Net // ¯\_(ツ)_/¯
)

type FreteRequest struct {
	CepOrigem        string
	CepDestino       string
	PesoKg           decimal.Decimal
	ComprimentoCm    decimal.Decimal
	AlturaCm         decimal.Decimal
	LarguraCm        decimal.Decimal
	Servicos         []TipoServico
	ValorDeclarado   decimal.Decimal
	AvisoRecebimento bool
	CdEmpresa        string
	DsSenha          string
	Mode             RequestMode
}

// SetServicos troca os tipos de serviço a serem consultados
func (r *FreteRequest) SetServicos(srvs ...TipoServico) *FreteRequest {
	r.Servicos = make([]TipoServico, 0)
	for _, v := range srvs {
		r.Servicos = append(r.Servicos, v)
	}
	return r
}

// AppendServico anexa o serviço srv aos serviços a serem consultados
func (r *FreteRequest) AppendServico(srv TipoServico) *FreteRequest {
	for _, v := range r.Servicos {
		if v == srv {
			// already exists
			return r
		}
	}
	r.Servicos = append(r.Servicos, srv)
	return r
}

// FreteResponse resposta dos correios
type FreteResponse struct {
	Servicos map[TipoServico]ServicoResponse
}

// Any retorna o primeiro serviço recebido
func (r *FreteResponse) Any() ServicoResponse {
	if r.Servicos == nil || len(r.Servicos) == 0 {
		return ServicoResponse{
			Erro:    &ServicoResponseError{Codigo: ErrIndeterminado},
			ErroMsg: "nenhum serviço encontrado",
		}
	}
	for _, v := range r.Servicos {
		return v
	}
	return ServicoResponse{
		Erro:    &ServicoResponseError{Codigo: ErrIndeterminado},
		ErroMsg: "nenhum serviço encontrado",
	}
}

// ServicoResponseError é a resposta de erro da API dos Correios
type ServicoResponseError struct {
	Codigo TipoErro
}

// ServicoResponse representa os dados retornados para um tipo de serviço
type ServicoResponse struct {
	Tipo                  TipoServico
	Preco                 decimal.Decimal // preço != valor != custo; deveria ser tudo preço
	PrazoEntregaDias      int
	PrecoSemAdicionais    decimal.Decimal
	PrecoMaoPropria       decimal.Decimal
	PrecoAvisoRecebimento decimal.Decimal
	PrecoValorDeclarado   decimal.Decimal
	EntregaDomiciliar     bool
	EntregaSabado         bool
	Erro                  *ServicoResponseError
	ErroMsg               string
}

// xml wrapper for ServicoResponse
type servicoResp struct {
	Codigo                string
	Valor                 string
	PrazoEntrega          int
	ValorSemAdicionais    string
	ValorMaoPropria       string
	ValorAvisoRecebimento string
	ValorValorDeclarado   string
	EntregaDomiciliar     string
	EntregaSabado         string
	Erro                  int
	MsgErro               string
}

// NewFreteRequest cria um struct *FreteRequest com os defaults:
//
// PesoKg         0.5
// ComprimentoCm 16.0
// LarguraCm     11.0
// AlturaCm       5.0
// Servicos      []{SvcSEDEXVarejo, SvcPACVarejo}
func NewFreteRequest(cepOrigem, cepDestino string) *FreteRequest {
	return &FreteRequest{
		CepOrigem:      cepOrigem,
		CepDestino:     cepDestino,
		PesoKg:         decimal.NewFromFloat(0.5),
		ComprimentoCm:  decimal.NewFromFloat(16.0),
		LarguraCm:      decimal.NewFromFloat(11.0),
		AlturaCm:       decimal.NewFromFloat(5.0),
		Servicos:       []TipoServico{SvcSEDEXVarejo, SvcPACVarejo},
		ValorDeclarado: decimal.NewFromFloat(0.0),
	}
}

// CalcularFrete envia o request p/ calcular o frete utilizando
// um *FreteRequest
// http://ws.correios.com.br/calculador/CalcPrecoPrazo.aspx?sCepOrigem=01243000&sCepDestino=04041002&nVlPeso=1&nCdFormato=1&nVlComprimento=16&nVlAltura=5&nVlLargura=11&StrRetorno=xml&nCdServico=40010,41106&nVlValorDeclarado=0
func CalcularFrete(ctx context.Context, req *FreteRequest) (*FreteResponse, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	// desde 2019, os Correios não aceitam consultas múltiplas caso não seja
	// informado o código da empresa + senha
	if len(req.Servicos) > 1 &&
		((req.Mode == RequestModeAuto && req.CdEmpresa == "") || (req.Mode == RequestModeSingle)) {
		reqs := make([]*FreteRequest, len(req.Servicos))
		for k := range req.Servicos {
			clone := &FreteRequest{
				CepOrigem:        req.CepOrigem,
				CepDestino:       req.CepDestino,
				PesoKg:           req.PesoKg,
				ComprimentoCm:    req.ComprimentoCm,
				AlturaCm:         req.AlturaCm,
				LarguraCm:        req.LarguraCm,
				Servicos:         []TipoServico{req.Servicos[k]},
				ValorDeclarado:   req.ValorDeclarado,
				AvisoRecebimento: req.AvisoRecebimento,
				CdEmpresa:        req.CdEmpresa,
				DsSenha:          req.DsSenha,
			}
			reqs[k] = clone
		}
		r00 := &FreteResponse{
			Servicos: make(map[TipoServico]ServicoResponse),
		}
		for i, v := range reqs {
			rsp, err := CalcularFrete(ctx, v)
			if err != nil && len(reqs) == i+1 {
				return r00, err
			} else if err != nil {
				continue
			}
			for k2, v2 := range rsp.Servicos {
				r00.Servicos[k2] = v2
			}
		}
		return r00, nil
	}
	v := url.Values{}
	v.Set("sCepOrigem", strings.Trim(req.CepOrigem, "-"))
	v.Set("sCepDestino", strings.Trim(req.CepDestino, "-"))
	v.Set("nVlPeso", req.PesoKg.String())
	v.Set("nCdFormato", "1")
	v.Set("nVlComprimento", req.ComprimentoCm.String())
	v.Set("nVlAltura", req.AlturaCm.String())
	v.Set("nVlLargura", req.LarguraCm.String())
	v.Set("StrRetorno", "xml")
	svcs := make([]string, len(req.Servicos))
	for k, v := range req.Servicos {
		svcs[k] = string(v)
	}
	v.Set("nCdServico", strings.Join(svcs, ","))
	v.Set("nVlValorDeclarado", req.ValorDeclarado.String())
	if req.AvisoRecebimento {
		v.Set("sCdAvisoRecebimento", "S")
	}
	if req.CdEmpresa != "" {
		v.Set("nCdEmpresa", req.CdEmpresa)
		v.Set("sDsSenha", req.DsSenha)
	}

	rq0, _ := http.NewRequest(http.MethodGet, FreteEndpoint+"?"+v.Encode(), nil)
	rq0 = rq0.WithContext(ctx)

	cresp, err := http.DefaultClient.Do(rq0)
	if err != nil {
		return nil, err
	}
	defer cresp.Body.Close()

	rrbuf := new(bytes.Buffer)
	io.Copy(rrbuf, cresp.Body)
	p := xml.NewDecoder(rrbuf)
	p.CharsetReader = CharsetReader

	vlov := struct {
		XMLName string        `xml:"Servicos"`
		Values  []servicoResp `xml:"cServico"`
	}{}

	err = p.Decode(&vlov)
	if err != nil {
		fmt.Println("CORREIOS: " + rrbuf.String())
		return nil, err
	}
	//
	output := &FreteResponse{
		Servicos: make(map[TipoServico]ServicoResponse),
	}
	//
	for _, v := range vlov.Values {
		v2 := ServicoResponse{}
		v2.Tipo = TipoServico(v.Codigo)
		v2.Preco, _ = decimal.NewFromString(fixWrongDecimals(v.Valor))
		v2.PrazoEntregaDias = v.PrazoEntrega
		v2.PrecoSemAdicionais, _ = decimal.NewFromString(fixWrongDecimals(v.ValorSemAdicionais))
		v2.PrecoMaoPropria, _ = decimal.NewFromString(fixWrongDecimals(v.ValorMaoPropria))
		v2.PrecoAvisoRecebimento, _ = decimal.NewFromString(fixWrongDecimals(v.ValorAvisoRecebimento))
		v2.PrecoValorDeclarado, _ = decimal.NewFromString(fixWrongDecimals(v.ValorValorDeclarado))
		v2.EntregaDomiciliar = (v.EntregaDomiciliar == "S")
		v2.EntregaSabado = (v.EntregaSabado == "S")
		if v.Erro != 0 {
			er9 := &ServicoResponseError{
				Codigo: TipoErro(v.Erro),
			}
			v2.Erro = er9
			v2.ErroMsg = v.MsgErro
		}
		output.Servicos[v2.Tipo] = v2
	}
	return output, nil
}

//////// dragons below
///

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
