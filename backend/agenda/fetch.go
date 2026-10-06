package agenda

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Limites do download de uma agenda. O link iCal privado é um segredo (dá
// leitura da agenda inteira): só https, poucos redirecionamentos (todos https)
// e nenhum erro devolvido contém a URL.
const (
	MaxICSBytes   = 10 << 20
	FetchTimeout  = 20 * time.Second
	maxRedirects  = 3
	maskTailChars = 6
)

// ErrInvalidURL indica um link que não é https (ou webcal) válido.
var ErrInvalidURL = errors.New("link de agenda inválido: use o endereço https (ou webcal://) do iCal")

// NormalizeURL apara o link, troca webcal:// por https:// e exige https com host.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", ErrInvalidURL
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "webcal", "webcals":
		u.Scheme = "https"
	default:
		return "", ErrInvalidURL
	}
	u.Scheme = "https"
	return u.String(), nil
}

// MaskURL mostra só o host e os últimos caracteres do link, o suficiente para
// o usuário reconhecer a agenda sem expor o segredo.
func MaskURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "…"
	}
	resto := strings.TrimPrefix(u.String(), u.Scheme+"://"+u.Host)
	runes := []rune(resto)
	if len(runes) > maskTailChars {
		runes = runes[len(runes)-maskTailChars:]
	}
	return u.Hostname() + "…" + string(runes)
}

// Fetcher baixa agendas. Client nil usa um cliente padrão; o cliente recebido
// é copiado e ganha timeout e a política de redirecionamento.
type Fetcher struct {
	Client   *http.Client
	MaxBytes int64
}

// Fetch baixa o .ics. Os erros nunca citam a URL (ela só aparece mascarada).
func (f Fetcher) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	link, err := NormalizeURL(rawURL)
	if err != nil {
		return nil, err
	}
	masked := MaskURL(link)
	if ctx == nil {
		ctx = context.Background()
	}

	client := &http.Client{}
	if f.Client != nil {
		c := *f.Client
		client = &c
	}
	client.Timeout = FetchTimeout
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > maxRedirects {
			return errors.New("redirecionamentos demais")
		}
		if req.URL.Scheme != "https" {
			return errors.New("redirecionamento para endereço sem https recusado")
		}
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao preparar o download da agenda %s", masked)
	}
	req.Header.Set("Accept", "text/calendar, */*;q=0.5")
	req.Header.Set("User-Agent", "teamwork-logger")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar a agenda %s: %v", masked, unwrapURLError(err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("a agenda %s respondeu HTTP %d", masked, resp.StatusCode)
	}

	limite := f.MaxBytes
	if limite <= 0 {
		limite = MaxICSBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limite+1))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler a agenda %s: %v", masked, unwrapURLError(err))
	}
	if int64(len(data)) > limite {
		return nil, fmt.Errorf("a agenda %s passa do limite de %d MB", masked, limite>>20)
	}
	return data, nil
}

// unwrapURLError tira a URL da mensagem de erro do net/http ("Get \"https://...\": ...").
func unwrapURLError(err error) error {
	var ue *url.Error
	for errors.As(err, &ue) {
		err = ue.Err
	}
	return err
}
