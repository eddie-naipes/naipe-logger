// Package notify define o contrato das notificações do sistema usado pelos
// lembretes e pelo cronômetro. O envio real (runtime do Wails) fica em
// backend/app_notifications.go; aqui só há tipos e um Sender falso, para que
// os testes nunca disparem notificações na máquina de quem os roda.
package notify

import "sync"

// DefaultAction é o ActionIdentifier que o Wails informa quando o usuário
// clica no corpo da notificação, e não num botão.
const DefaultAction = "DEFAULT_ACTION"

// Action é um botão da notificação.
type Action struct {
	ID    string
	Title string
}

// Category agrupa os botões de um tipo de notificação. Precisa ser registrada
// antes de enviar uma notificação que a use.
type Category struct {
	ID      string
	Actions []Action
}

// Notification é o que se pede para mostrar. Data volta intacto na resposta,
// e é por ele que a resposta é roteada (campo "kind").
type Notification struct {
	ID         string
	Title      string
	Body       string
	CategoryID string
	Data       map[string]any
}

// Response é a interação do usuário com uma notificação.
type Response struct {
	ActionID string
	Data     map[string]any
}

// Kind devolve Data["kind"] como texto ("" se ausente).
func (r Response) Kind() string { return r.String("kind") }

// String devolve Data[key] como texto ("" se ausente ou de outro tipo).
func (r Response) String(key string) string {
	if v, ok := r.Data[key].(string); ok {
		return v
	}
	return ""
}

// Sender envia notificações.
type Sender interface {
	Send(n Notification) error
}

// Recorder é um Sender falso que só guarda o que receberia. Err, se definido,
// é devolvido a cada envio.
type Recorder struct {
	mu   sync.Mutex
	sent []Notification
	Err  error
}

// Send registra a notificação.
func (r *Recorder) Send(n Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, n)
	return r.Err
}

// Sent devolve uma cópia do que foi enviado.
func (r *Recorder) Sent() []Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Notification(nil), r.sent...)
}
