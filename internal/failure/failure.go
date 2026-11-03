package failure

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	Context     Kind = "context"
	Model       Kind = "model"
	Quota       Kind = "quota"
	Auth        Kind = "auth"
	Refused     Kind = "refused"
	Launch      Kind = "launch"
	Transient   Kind = "transient"
	Timeout     Kind = "timeout"
	Interrupted Kind = "interrupted"
	Unknown     Kind = "unknown"
)

var RetryWaits = []time.Duration{20 * time.Second, 60 * time.Second, 120 * time.Second}

const Heartbeat = 10 * time.Second
const IdleKill = 300 * time.Second
const (
	questionLineLimit = 3
	questionRuneLimit = 500
	shortWaitPadding  = time.Minute
	longWaitPadding   = 5 * time.Minute
)

var DefaultSwitch = map[Kind]time.Duration{Quota: 24 * time.Hour, Transient: time.Hour, Timeout: time.Hour, Refused: time.Hour}
var patterns = []struct {
	k Kind
	r *regexp.Regexp
}{
	{"context", regexp.MustCompile(`(?i)context (length|window)|maximum context|prompt is too long|too many tokens|input.{0,20}too long|exceeds the (maximum|context)`)},
	{"model", regexp.MustCompile(`(?i)model[_ ]?not[_ ]?found|ProviderModelNotFound|unknown model|invalid model|no such model|model.{0,40}(does not exist|is not supported|not supported)`)},
	{"quota", regexp.MustCompile(`(?i)\b429\b|RESOURCE_EXHAUSTED|TerminalQuotaError|rate.?limit|quota|usage limit|limit reached|hit your (usage )?limit|weekly limit|5-hour limit|too many requests|insufficient_quota|insufficient (funds|credits)|credit balance|billing|payment required|\b402\b|free usage exceeded|out of (credits|tokens)|exhausted your`)},
	{"auth", regexp.MustCompile(`(?i)not running in a trusted director|untrusted (folder|workspace)|\b401\b|\b403\b|unauthori[sz]ed|invalid api key|api key (is )?(not set|missing|invalid)|not logged in|login required|please (log ?in|sign ?in|run /login)|not authenticated|authentication (failed|error|required)`)},
	{"refused", regexp.MustCompile(`(?i)content (policy|filter)|blocked by|PROHIBITED_CONTENT|violates .{0,25}polic|I can.?t (help|assist) with`)},
	{"launch", regexp.MustCompile(`(?i)não pode encontrar o arquivo|cannot find the (file|path) specified|is not recognized as an internal or external command|command not found`)},
	{"transient", regexp.MustCompile(`(?i)\b50[234]\b|\b529\b|UNAVAILABLE|overloaded|at capacity|high demand|bad gateway|service unavailable|server error|ECONNRESET|ECONNREFUSED|ETIMEDOUT|ENOTFOUND|EAI_AGAIN|getaddrinfo|socket hang up|fetch failed|connection (error|reset|refused|closed)|network (error|is unreachable)|timed? ?out|temporarily|try again later|internal (server )?error`)},
}
var help = map[Kind]string{
	"transient":   "O serviço do modelo está instável ou sobrecarregado (erro temporário).",
	"timeout":     "A ferramenta ficou muda por tempo demais (timeout) e foi encerrada.",
	"quota":       "Cota, limite de uso ou créditos do modelo esgotados.",
	"auth":        "A ferramenta não está autenticada ou a chave foi recusada.",
	"model":       "O modelo pedido não existe ou não está liberado para a sua conta.",
	"context":     "O pedido ficou grande demais para a janela de contexto do modelo.",
	"refused":     "O modelo recusou o pedido (filtro de conteúdo ou política).",
	"interrupted": "A sessão foi interrompida por você (Ctrl+C).",
	"launch":      "A ferramenta não iniciou (não está instalada ou não está no PATH).",
	"unknown":     "A ferramenta terminou com erro sem um motivo reconhecido. Veja o log.",
}
var actions = map[Kind]string{
	"quota":   "Espere a cota voltar, troque de modelo ou de ferramenta, ou adicione créditos.",
	"auth":    "Entre de novo no cliente (login ou chave) ou marque a pasta como confiável, e mande a instrução de novo.",
	"model":   "Escolha outro modelo nas propriedades do agente (Organograma) ou corrija o catálogo do cliente.",
	"context": "Mande uma instrução menor ou divida a tarefa em partes.",
	"refused": "Reescreva a instrução com mais contexto, ou troque de modelo.",
	"launch":  "Instale o cliente ou confira o PATH (aba Clientes).",
}

func Classify(exitCode int, text string) Kind {
	switch exitCode {
	case 0:
		return ""
	case 124:
		return Timeout
	case 130:
		return Interrupted
	}
	for _, p := range patterns {
		if p.r.MatchString(text) {
			return p.k
		}
	}
	return Unknown
}
func Help(k Kind) string    { return help[k] }
func Action(k Kind) string  { return actions[k] }
func AutoRetry(k Kind) bool { return k == Transient || k == Timeout }
func Switchable(k Kind) bool {
	return k == Quota || k == Transient || k == Timeout || k == Model || k == Refused || k == Launch
}
func AskRetry(k Kind) bool { return AutoRetry(k) || k == Unknown }

var waitPrefix = regexp.MustCompile(`(?i)(?:retry|try again|resets?)\s+(?:in|after)\s+`)
var unit = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(hours?|hrs?|h|minutes?|mins?|m|seconds?|secs?|s)`)
var clock12 = regexp.MustCompile(`(?i)(?:resets?|try again|available again)(?:\s+at)?\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)\b`)
var clock24 = regexp.MustCompile(`(?i)(?:resets?|try again|available again)(?:\s+at)?\s+(\d{1,2}):(\d{2})()`)

func ParseWait(text string, now time.Time) (time.Duration, bool) {
	for _, loc := range waitPrefix.FindAllStringIndex(text, -1) {
		tail := text[loc[1]:]
		var total float64
		found := false
		for {
			m := unit.FindStringSubmatchIndex(tail)
			if m == nil {
				break
			}
			end := m[1]
			// Substitui o lookahead: inspeciona a próxima letra sem consumir o próximo componente.
			if end < len(tail) && ((tail[end] >= 'a' && tail[end] <= 'z') || (tail[end] >= 'A' && tail[end] <= 'Z')) {
				break
			}
			n, _ := strconv.ParseFloat(tail[m[2]:m[3]], 64)
			u := strings.ToLower(tail[m[4]:m[5]])[0]
			factor := float64(1)
			if u == 'h' {
				factor = 3600
			} else if u == 'm' {
				factor = 60
			}
			total += n * factor
			found = true
			tail = strings.TrimSpace(tail[end:])
			if strings.HasPrefix(strings.ToLower(tail), "and ") {
				tail = strings.TrimSpace(tail[4:])
			}
		}
		if found && total < float64(int64(1<<63-1))/float64(time.Second) {
			return time.Duration(total * float64(time.Second)), true
		}
	}
	m := clock12.FindStringSubmatch(text)
	if m == nil {
		m = clock24.FindStringSubmatch(text)
	}
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	minute, _ := strconv.Atoi(m[2])
	if minute > 59 {
		return 0, false
	}
	if m[3] != "" {
		h %= 12
		if strings.EqualFold(m[3], "pm") {
			h += 12
		}
	}
	at := time.Date(now.Year(), now.Month(), now.Day(), h%24, minute, 0, 0, now.Location())
	if !at.After(now) {
		at = at.AddDate(0, 0, 1)
	}
	return at.Sub(now), true
}
func RetryHint(text string, now time.Time) string {
	d, ok := ParseWait(text, now)
	if !ok {
		return ""
	}
	s := int64(d / time.Second)
	if s >= 3600 {
		return fmt.Sprintf("%dh%dm", s/3600, s%3600/60)
	}
	if s >= 60 {
		return fmt.Sprintf("%dm", s/60)
	}
	return fmt.Sprintf("%ds", s)
}

var hintPattern = regexp.MustCompile(`(\d+)([hms])`)

func HintDuration(hint string) (time.Duration, bool) {
	var s time.Duration
	for _, m := range hintPattern.FindAllStringSubmatch(hint, -1) {
		n, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return 0, false
		}
		f := time.Second
		if m[2] == "h" {
			f = time.Hour
		} else if m[2] == "m" {
			f = time.Minute
		}
		if n > int64((time.Duration(1<<63-1)-s)/f) {
			return 0, false
		}
		s += time.Duration(n) * f
	}
	if s == 0 {
		return 0, false
	}
	pad := shortWaitPadding
	if s >= time.Hour {
		pad = longWaitPadding
	}
	if s > time.Duration(1<<63-1)-pad {
		return 0, false
	}
	return s + pad, true
}
func FormatDuration(d time.Duration) string {
	s := int64(d / time.Second)
	if s >= 3600 {
		return fmt.Sprintf("%dh%02dm", s/3600, s%3600/60)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}

var tokensPattern = regexp.MustCompile(`(?i)tokens used\s*[:\n]\s*([\d.,]+)`)

func ParseTokensUsed(text string) (int, bool) {
	m := tokensPattern.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	n, e := strconv.Atoi(strings.NewReplacer(",", "", ".", "").Replace(m[1]))
	return n, e == nil
}
func LastQuestion(text string) string {
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 || !strings.HasSuffix(lines[len(lines)-1], "?") {
		return ""
	}
	if len(lines) > questionLineLimit {
		lines = lines[len(lines)-questionLineLimit:]
	}
	r := []rune(strings.Join(lines, " "))
	if len(r) > questionRuneLimit {
		r = r[len(r)-questionRuneLimit:]
	}
	return string(r)
}

var modelPattern = regexp.MustCompile(`\bmodel:\s*([A-Za-z0-9._/\-]+)`)

func ErrorModel(text string) string {
	m := modelPattern.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}
