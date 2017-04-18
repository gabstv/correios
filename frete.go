package correios

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

type TipoServico string

type TipoErro int

const (
	//https://www.correios.com.br/para-voce/correios-de-a-a-z/pdf/calculador-remoto-de-precos-e-prazos/manual-de-implementacao-do-calculo-remoto-de-precos-e-prazos
	SEDEX_Varejo          TipoServico = "40010"
	SEDEX_a_Cobrar_Varejo TipoServico = "40045"
	SEDEX_10_Varejo       TipoServico = "40215"
	SEDEX_Hoje_Varejo     TipoServico = "40290"
	PAC_Varejo            TipoServico = "41106"
	//
	ERR_TipoServicoInvalido          TipoErro = -1
	ERR_CepOrigemInvalido            TipoErro = -2
	ERR_CepDestinoInvalido           TipoErro = -3
	ERR_CepPesoExcedido              TipoErro = -4
	ERR_ValorDeclaradoAlto10k        TipoErro = -5
	ERR_ServicoIndisponivelTrecho    TipoErro = -6
	ERR_ValorDeclaradoObrigatorio    TipoErro = -7
	ERR_MaoPropriaIndisponivel       TipoErro = -8
	ERR_AvisoRecebimentoIndisponivel TipoErro = -9
	ERR_PrecificacaoIndisponivel     TipoErro = -10
	ERR_InformarDimensoes            TipoErro = -11
	ERR_Comprimento                  TipoErro = -12
	ERR_Largura                      TipoErro = -13
	ERR_Altura                       TipoErro = -14
	ERR_Comprimento105               TipoErro = -15  // > 105cm
	ERR_Largura105                   TipoErro = -16  // > 105cm
	ERR_Altura105                    TipoErro = -17  // > 105cm
	ERR_AlturaInferior               TipoErro = -18  // < 2cm
	ERR_LarguraInferior              TipoErro = -20  // < 11cm
	ERR_ComprimentoInferior          TipoErro = -22  // < 16cm
	ERR_DimensoesSoma                TipoErro = -23  // A soma resultante do comprimento + largura + altura não deve superar a 200 cm
	ERR_Comprimento2                 TipoErro = -24  // WTF (ver -12)
	ERR_Diametro                     TipoErro = -25  // Diâmetro inválido
	ERR_Comprimento3                 TipoErro = -26  // WTF (ver -12)
	ERR_Diametro2                    TipoErro = -27  // ?
	ERR_Comprimento4                 TipoErro = -28  // O comprimento não pode ser maior que 105 cm.
	ERR_Diametro91                   TipoErro = -29  // O diâmetro não pode ser maior que 91 cm.
	ERR_Comprimento18                TipoErro = -30  // O comprimento não pode ser inferior a 18 cm.
	ERR_Diametro5                    TipoErro = -31  // O diâmetro não pode ser inferior a 5 cm.
	ERR_SomaDiametro                 TipoErro = -32  // A soma resultante do comprimento + o dobro do diâmetro não deve superar a 200 cm
	ERR_SistemaIndisponivel          TipoErro = -33  // Sistema temporariamente fora do ar. Favor tentar mais tarde.
	ERR_CodigoOuSenha                TipoErro = -34  // Código Administrativo ou Senha inválidos.
	ERR_Senha                        TipoErro = -35  // Senha incorreta.
	ERR_SemContrato                  TipoErro = -36  // Cliente não possui contrato vigente com os Correios.
	ERR_SemServicoAtivo              TipoErro = -37  // Cliente não possui serviço ativo em seu contrato.
	ERR_ServicoIndisponivelAdmin     TipoErro = -38  // Serviço indisponível para este código administrativo.
	ERR_PesoExcedidoEnvelope         TipoErro = -39  // Peso excedido para o formato envelope
	ERR_InformarDimensoes2           TipoErro = -40  // Para definicao do preco deverao ser informados, tambem, o comprimento e a largura e altura do objeto em centimetros (cm).
	ERR_Comprimento60                TipoErro = -41  // O comprimento nao pode ser maior que 60 cm.
	ERR_Comprimento16                TipoErro = -42  // (repetido) O comprimento nao pode ser inferior a 16 cm.
	ERR_ComprimentoLargura120        TipoErro = -43  // A soma resultante do comprimento + largura nao deve superar a 120 cm
	ERR_LarguraInferior2             TipoErro = -44  // < 11cm
	ERR_LarguraSuperior60            TipoErro = -44  // > 60cm
	ERR_ErroCalculoTarifa            TipoErro = -888 // Erro ao calcular a tarifa
	ERR_LocalidadeOrigem             TipoErro = 006  // Localidade de origem não abrange o serviço informado
	ERR_LocalidadeDestino            TipoErro = 007  // Localidade de destino não abrange o serviço informado
	ERR_ServicoIndisponivelTrecho2   TipoErro = 8    // 008 Serviço indisponível para o trecho informado
	ERR_AreaDeRiscoCEPInicial        TipoErro = 9    // 009 CEP inicial pertencente a Área de Risco.
	ERR_AreaPrazoDiferenciado        TipoErro = 010  // Área com entrega temporariamente sujeita a prazo diferenciado.
	ERR_AreaDeRiscoCEPs              TipoErro = 011  // CEP inicial e final pertencentes a Área de Risco
	ERR_Indisponivel                 TipoErro = 7    // Serviço indisponível, tente mais tarde
	ERR_WTF                          TipoErro = 99   // Outros erros diversos do .Net // ¯\_(ツ)_/¯
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
}

type FreteResponse struct {
	Servicos map[TipoServico]ServicoResponse
}

type ServicoResponseError struct {
	Codigo TipoErro
}

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

func NewFreteRequest(cepOrigem, cepDestino string) *FreteRequest {
	return &FreteRequest{
		CepOrigem:      cepOrigem,
		CepDestino:     cepDestino,
		PesoKg:         decimal.NewFromFloat(0.5),
		ComprimentoCm:  decimal.NewFromFloat(16.0),
		LarguraCm:      decimal.NewFromFloat(11.0),
		AlturaCm:       decimal.NewFromFloat(5.0),
		Servicos:       []TipoServico{SEDEX_Varejo, PAC_Varejo},
		ValorDeclarado: decimal.NewFromFloat(0.0),
	}
}

// http://ws.correios.com.br/calculador/CalcPrecoPrazo.aspx?sCepOrigem=01243000&sCepDestino=04041002&nVlPeso=1&nCdFormato=1&nVlComprimento=16&nVlAltura=5&nVlLargura=11&StrRetorno=xml&nCdServico=40010,41106&nVlValorDeclarado=0
func CalcularFrete(req *FreteRequest) (*FreteResponse, error) {
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

	cresp, err := http.Get("http://ws.correios.com.br/calculador/CalcPrecoPrazo.aspx?" + v.Encode())
	if err != nil {
		return nil, err
	}
	defer cresp.Body.Close()

	p := xml.NewDecoder(cresp.Body)
	p.CharsetReader = CharsetReader

	vlov := struct {
		XMLName string        `xml:"Servicos"`
		Values  []servicoResp `xml:"cServico"`
	}{}

	err = p.Decode(&vlov)
	if err != nil {
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
