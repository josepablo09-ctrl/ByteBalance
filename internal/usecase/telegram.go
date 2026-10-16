package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// ErrTelegramNotLinked indica que el usuario no tiene un chat de Telegram vinculado.
var ErrTelegramNotLinked = errors.New("la cuenta no tiene Telegram vinculado")

// linkCodePattern es el formato de los códigos que genera user.NewLinkCode. Se
// valida antes de consultar la base para descartar texto arbitrario.
var linkCodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Mensajes del bot. Se envían como texto plano (sin parse_mode), así ningún
// contenido se interpreta como formato.
const (
	msgHelp = "Hola, soy el bot de ByteBalance.\n\n" +
		"Para vincular tu cuenta abre ByteBalance, ve a Ajustes y pulsa «Conectar Telegram».\n" +
		"Envía /stop para dejar de recibir mensajes."
	msgPrivateOnly   = "Solo puedo vincular cuentas desde un chat privado conmigo."
	msgInvalidCode   = "Este enlace venció o ya se usó. Genera uno nuevo en ByteBalance › Ajustes."
	msgChatTaken     = "Este chat ya está vinculado a otra cuenta de ByteBalance. Envía /stop, genera un enlace nuevo en Ajustes y vuelve a abrirlo."
	msgAlreadyLinked = "Este chat ya está vinculado a tu cuenta de ByteBalance."
	msgNotLinked     = "Este chat no está vinculado a ninguna cuenta de ByteBalance."
	msgUnlinked      = "Listo: desvinculé este chat de ByteBalance. Ya no recibirás mensajes."
	msgTest          = "Mensaje de prueba de ByteBalance: las notificaciones llegan bien a este chat."
)

// TelegramSender envía mensajes de texto a un chat de Telegram.
type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

// TelegramLink es el enlace para vincular la cuenta con el bot.
type TelegramLink struct {
	// Code es el código de un solo uso; también puede enviarse como "/start <code>".
	Code      string
	URL       string
	ExpiresAt time.Time
}

// TelegramService vincula cuentas con chats de Telegram y atiende los mensajes
// que recibe el bot.
type TelegramService struct {
	users       user.Repository
	codes       user.LinkCodeRepository
	bot         TelegramSender
	botUsername string
	now         func() time.Time
}

// NewTelegramService crea un TelegramService para el bot botUsername. Si now
// es nil se usa time.Now.
func NewTelegramService(users user.Repository, codes user.LinkCodeRepository, bot TelegramSender, botUsername string, now func() time.Time) *TelegramService {
	return &TelegramService{users: users, codes: codes, bot: bot, botUsername: botUsername, now: clockOrDefault(now)}
}

// BotUsername devuelve el nombre de usuario del bot, sin "@".
func (s *TelegramService) BotUsername() string {
	return s.botUsername
}

// CreateLink genera un código de vinculación para el usuario e invalida los
// anteriores. El enlace abre el bot con el código como parámetro start.
func (s *TelegramService) CreateLink(ctx context.Context, userID domain.ID) (TelegramLink, error) {
	if _, err := s.users.GetByID(ctx, userID); err != nil {
		return TelegramLink{}, fmt.Errorf("crear enlace de Telegram: %w", err)
	}
	code, lc, err := user.NewLinkCode(userID, s.now())
	if err != nil {
		return TelegramLink{}, fmt.Errorf("crear enlace de Telegram: %w", err)
	}
	if err := s.codes.Replace(ctx, lc); err != nil {
		return TelegramLink{}, fmt.Errorf("crear enlace de Telegram: %w", err)
	}
	return TelegramLink{
		Code:      code,
		URL:       "https://t.me/" + s.botUsername + "?start=" + code,
		ExpiresAt: lc.ExpiresAt,
	}, nil
}

// Unlink desvincula el chat de Telegram del usuario y avisa en ese chat. Es
// idempotente: si no había chat vinculado no hace nada.
func (s *TelegramService) Unlink(ctx context.Context, userID domain.ID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("desvincular Telegram: %w", err)
	}
	if !u.HasTelegram() {
		return nil
	}
	chatID := *u.TelegramChatID
	u.UnlinkTelegram()
	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("desvincular Telegram: %w", err)
	}
	// El aviso es una cortesía: si falla, la desvinculación ya quedó hecha.
	if err := s.bot.SendMessage(ctx, chatID, msgUnlinked); err != nil {
		slog.WarnContext(ctx, "avisar desvinculación por Telegram", "error", err)
	}
	return nil
}

// SendTest envía un mensaje de prueba al chat vinculado del usuario.
// Devuelve ErrTelegramNotLinked si no tiene Telegram vinculado.
func (s *TelegramService) SendTest(ctx context.Context, userID domain.ID) error {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("enviar mensaje de prueba: %w", err)
	}
	if !u.HasTelegram() {
		return ErrTelegramNotLinked
	}
	if err := s.bot.SendMessage(ctx, *u.TelegramChatID, msgTest); err != nil {
		return fmt.Errorf("enviar mensaje de prueba: %w", err)
	}
	return nil
}

// HandleMessage atiende un mensaje recibido por el bot y responde en el chat:
// "/start <código>" vincula la cuenta, "/stop" la desvincula y cualquier otro
// texto recibe la ayuda. Los mensajes de grupos y canales se ignoran.
func (s *TelegramService) HandleMessage(ctx context.Context, chatID int64, chatType, text string) error {
	command, arg := parseCommand(text)
	if chatType != "private" {
		if command == "/start" {
			return s.reply(ctx, chatID, msgPrivateOnly)
		}
		return nil
	}
	switch command {
	case "/start":
		if arg == "" {
			return s.reply(ctx, chatID, msgHelp)
		}
		return s.link(ctx, chatID, arg)
	case "/stop":
		return s.unlinkChat(ctx, chatID)
	default:
		return s.reply(ctx, chatID, msgHelp)
	}
}

func (s *TelegramService) link(ctx context.Context, chatID int64, code string) error {
	if !linkCodePattern.MatchString(code) {
		return s.reply(ctx, chatID, msgInvalidCode)
	}
	userID, err := s.codes.Consume(ctx, user.HashLinkCode(code), s.now())
	if errors.Is(err, domain.ErrNotFound) {
		return s.reply(ctx, chatID, msgInvalidCode)
	}
	if err != nil {
		return fmt.Errorf("vincular Telegram: %w", err)
	}

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("vincular Telegram: %w", err)
	}
	if u.TelegramChatID != nil && *u.TelegramChatID == chatID {
		return s.reply(ctx, chatID, msgAlreadyLinked)
	}
	if err := u.LinkTelegram(chatID); err != nil {
		return fmt.Errorf("vincular Telegram: %w", err)
	}
	if err := s.users.Update(ctx, u); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return s.reply(ctx, chatID, msgChatTaken)
		}
		return fmt.Errorf("vincular Telegram: %w", err)
	}
	return s.reply(ctx, chatID, linkedMessage(u.FullName))
}

func (s *TelegramService) unlinkChat(ctx context.Context, chatID int64) error {
	u, err := s.users.GetByTelegramChatID(ctx, chatID)
	if errors.Is(err, domain.ErrNotFound) {
		return s.reply(ctx, chatID, msgNotLinked)
	}
	if err != nil {
		return fmt.Errorf("desvincular Telegram: %w", err)
	}
	u.UnlinkTelegram()
	if err := s.users.Update(ctx, u); err != nil {
		return fmt.Errorf("desvincular Telegram: %w", err)
	}
	return s.reply(ctx, chatID, msgUnlinked)
}

func (s *TelegramService) reply(ctx context.Context, chatID int64, text string) error {
	if err := s.bot.SendMessage(ctx, chatID, text); err != nil {
		return fmt.Errorf("responder en Telegram: %w", err)
	}
	return nil
}

// linkedMessage confirma la vinculación usando solo el primer nombre.
func linkedMessage(fullName string) string {
	firstName, _, _ := strings.Cut(strings.TrimSpace(fullName), " ")
	return "Listo, " + firstName + ": tu cuenta de ByteBalance quedó vinculada. " +
		"Aquí recibirás los recordatorios de tus hábitos.\n\nEnvía /stop para desvincular este chat."
}

// parseCommand separa un comando de Telegram de su argumento. Acepta la forma
// "/start@NombreDelBot código" que Telegram usa en algunos clientes.
func parseCommand(text string) (command, arg string) {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return "", ""
	}
	command, _, _ = strings.Cut(strings.ToLower(fields[0]), "@")
	if len(fields) > 1 {
		arg = fields[1]
	}
	return command, arg
}
