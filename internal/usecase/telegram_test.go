package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// sentMessage es un mensaje enviado por el fakeBot.
type sentMessage struct {
	chatID int64
	text   string
}

// fakeBot registra los mensajes enviados; con err falla todos los envíos.
type fakeBot struct {
	sent []sentMessage
	err  error
}

func (b *fakeBot) SendMessage(_ context.Context, chatID int64, text string) error {
	if b.err != nil {
		return b.err
	}
	b.sent = append(b.sent, sentMessage{chatID, text})
	return nil
}

func (b *fakeBot) last(t *testing.T) sentMessage {
	t.Helper()
	if len(b.sent) == 0 {
		t.Fatal("el bot no envió ningún mensaje")
	}
	return b.sent[len(b.sent)-1]
}

// fakeLinkCodes es un user.LinkCodeRepository en memoria.
type fakeLinkCodes struct {
	byHash map[string]*storedCode
}

type storedCode struct {
	code user.LinkCode
	used bool
}

var _ user.LinkCodeRepository = (*fakeLinkCodes)(nil)

func newFakeLinkCodes() *fakeLinkCodes {
	return &fakeLinkCodes{byHash: map[string]*storedCode{}}
}

func (f *fakeLinkCodes) Replace(_ context.Context, c *user.LinkCode) error {
	for hash, stored := range f.byHash {
		if stored.code.UserID == c.UserID {
			delete(f.byHash, hash)
		}
	}
	f.byHash[string(c.Hash)] = &storedCode{code: *c}
	return nil
}

func (f *fakeLinkCodes) Consume(_ context.Context, hash []byte, now time.Time) (domain.ID, error) {
	stored, ok := f.byHash[string(hash)]
	if !ok || stored.used || !now.Before(stored.code.ExpiresAt) {
		return domain.ID{}, domain.ErrNotFound
	}
	stored.used = true
	return stored.code.UserID, nil
}

const chatA, chatB = int64(1001), int64(2002)

func TestTelegramServiceCreateLink(t *testing.T) {
	u := existingUser(t, "ana@example.com", "UTC")
	codes := newFakeLinkCodes()
	svc := NewTelegramService(newFakeUserRepo(u), codes, &fakeBot{}, "ByteBalanceBot", fixedClock)

	link, err := svc.CreateLink(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("CreateLink() error inesperado: %v", err)
	}
	if link.URL != "https://t.me/ByteBalanceBot?start="+link.Code || !link.ExpiresAt.Equal(fixedNow.Add(user.LinkCodeTTL)) {
		t.Errorf("enlace = %+v", link)
	}
	if _, ok := codes.byHash[string(user.HashLinkCode(link.Code))]; !ok || len(codes.byHash) != 1 {
		t.Error("debe guardarse solo el hash del código")
	}
	if _, err := svc.CreateLink(context.Background(), domain.NewID()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("usuario inexistente: error = %v, se esperaba ErrNotFound", err)
	}
}

func TestTelegramServiceLinkFlow(t *testing.T) {
	ctx := context.Background()

	t.Run("/start con un código válido vincula el chat", func(t *testing.T) {
		u := existingUser(t, "ana@example.com", "UTC")
		users, bot := newFakeUserRepo(u), &fakeBot{}
		svc := NewTelegramService(users, newFakeLinkCodes(), bot, "Bot", fixedClock)
		link, err := svc.CreateLink(ctx, u.ID)
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}

		if err := svc.HandleMessage(ctx, chatA, "private", "/start "+link.Code); err != nil {
			t.Fatalf("HandleMessage() error inesperado: %v", err)
		}
		if got := users.byID[u.ID].TelegramChatID; got == nil || *got != chatA {
			t.Fatalf("TelegramChatID = %v, se esperaba %d", got, chatA)
		}
		if msg := bot.last(t); msg.chatID != chatA || !strings.Contains(msg.text, "Listo, Ana") {
			t.Errorf("respuesta = %+v", msg)
		}

		// El mismo código no sirve dos veces.
		if err := svc.HandleMessage(ctx, chatB, "private", "/start "+link.Code); err != nil {
			t.Fatalf("segundo /start: %v", err)
		}
		if bot.last(t).text != msgInvalidCode || *users.byID[u.ID].TelegramChatID != chatA {
			t.Error("un código usado no debe volver a vincular")
		}
	})

	t.Run("código vencido", func(t *testing.T) {
		u := existingUser(t, "ana@example.com", "UTC")
		now := fixedNow
		users, bot := newFakeUserRepo(u), &fakeBot{}
		svc := NewTelegramService(users, newFakeLinkCodes(), bot, "Bot", func() time.Time { return now })
		link, err := svc.CreateLink(ctx, u.ID)
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}
		now = now.Add(user.LinkCodeTTL)

		if err := svc.HandleMessage(ctx, chatA, "private", "/start "+link.Code); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if bot.last(t).text != msgInvalidCode || isLinked(users, u.ID) {
			t.Error("un código vencido no debe vincular")
		}
	})

	t.Run("chat ya vinculado a otra cuenta", func(t *testing.T) {
		owner := existingUser(t, "ana@example.com", "UTC")
		other := existingUser(t, "otro@example.com", "UTC")
		if err := other.LinkTelegram(chatA); err != nil {
			t.Fatal(err)
		}
		users, bot := newFakeUserRepo(owner, other), &fakeBot{}
		svc := NewTelegramService(users, newFakeLinkCodes(), bot, "Bot", fixedClock)
		link, err := svc.CreateLink(ctx, owner.ID)
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}

		if err := svc.HandleMessage(ctx, chatA, "private", "/start "+link.Code); err != nil {
			t.Fatalf("HandleMessage: %v", err)
		}
		if bot.last(t).text != msgChatTaken || isLinked(users, owner.ID) {
			t.Error("no se debe robar el chat de otra cuenta")
		}
	})
}

func TestTelegramServiceHandleMessage(t *testing.T) {
	ctx := context.Background()
	linked := existingUser(t, "ana@example.com", "UTC")
	if err := linked.LinkTelegram(chatA); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		chatID    int64
		chatType  string
		text      string
		wantReply string // vacío = no responde
		wantLink  bool   // el usuario sigue vinculado al terminar
	}{
		{name: "/start sin código muestra la ayuda", chatID: chatB, chatType: "private", text: "/start", wantReply: msgHelp, wantLink: true},
		{name: "texto libre muestra la ayuda", chatID: chatB, chatType: "private", text: "hola", wantReply: msgHelp, wantLink: true},
		{name: "código con caracteres inválidos", chatID: chatB, chatType: "private", text: "/start <script>", wantReply: msgInvalidCode, wantLink: true},
		{name: "código inexistente", chatID: chatB, chatType: "private", text: "/start abcDEF123_-", wantReply: msgInvalidCode, wantLink: true},
		{name: "/stop desde el chat vinculado", chatID: chatA, chatType: "private", text: "/stop", wantReply: msgUnlinked, wantLink: false},
		{name: "/stop con mención al bot", chatID: chatA, chatType: "private", text: "/stop@ByteBalanceBot", wantReply: msgUnlinked, wantLink: false},
		{name: "/stop desde un chat sin vincular", chatID: chatB, chatType: "private", text: "/stop", wantReply: msgNotLinked, wantLink: true},
		{name: "/start en un grupo", chatID: -500, chatType: "group", text: "/start abc", wantReply: msgPrivateOnly, wantLink: true},
		{name: "otros mensajes de grupo se ignoran", chatID: -500, chatType: "group", text: "hola a todos", wantLink: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, bot := newFakeUserRepo(linked), &fakeBot{}
			svc := NewTelegramService(users, newFakeLinkCodes(), bot, "ByteBalanceBot", fixedClock)

			if err := svc.HandleMessage(ctx, tt.chatID, tt.chatType, tt.text); err != nil {
				t.Fatalf("HandleMessage() error inesperado: %v", err)
			}
			switch {
			case tt.wantReply == "" && len(bot.sent) > 0:
				t.Errorf("no debía responder, respondió %q", bot.sent[0].text)
			case tt.wantReply != "":
				if msg := bot.last(t); msg.text != tt.wantReply || msg.chatID != tt.chatID {
					t.Errorf("respuesta = %+v, se esperaba %q", msg, tt.wantReply)
				}
			}
			if got := isLinked(users, linked.ID); got != tt.wantLink {
				t.Errorf("vinculado = %v, se esperaba %v", got, tt.wantLink)
			}
		})
	}
}

func TestTelegramServiceUnlinkAndTest(t *testing.T) {
	ctx := context.Background()
	linked := existingUser(t, "ana@example.com", "UTC")
	if err := linked.LinkTelegram(chatA); err != nil {
		t.Fatal(err)
	}
	unlinked := existingUser(t, "otro@example.com", "UTC")

	t.Run("mensaje de prueba al chat vinculado", func(t *testing.T) {
		bot := &fakeBot{}
		svc := NewTelegramService(newFakeUserRepo(linked), newFakeLinkCodes(), bot, "Bot", fixedClock)
		if err := svc.SendTest(ctx, linked.ID); err != nil {
			t.Fatalf("SendTest: %v", err)
		}
		if msg := bot.last(t); msg.chatID != chatA || msg.text != msgTest {
			t.Errorf("mensaje = %+v", msg)
		}
	})

	t.Run("mensaje de prueba sin Telegram", func(t *testing.T) {
		svc := NewTelegramService(newFakeUserRepo(unlinked), newFakeLinkCodes(), &fakeBot{}, "Bot", fixedClock)
		if err := svc.SendTest(ctx, unlinked.ID); !errors.Is(err, ErrTelegramNotLinked) {
			t.Errorf("error = %v, se esperaba ErrTelegramNotLinked", err)
		}
	})

	t.Run("desvincular avisa en el chat y no falla si el aviso falla", func(t *testing.T) {
		users := newFakeUserRepo(linked)
		bot := &fakeBot{err: errors.New("telegram caído")}
		svc := NewTelegramService(users, newFakeLinkCodes(), bot, "Bot", fixedClock)
		if err := svc.Unlink(ctx, linked.ID); err != nil {
			t.Fatalf("Unlink: %v", err)
		}
		if isLinked(users, linked.ID) {
			t.Error("debía quedar desvinculado")
		}
	})

	t.Run("desvincular es idempotente", func(t *testing.T) {
		bot := &fakeBot{}
		svc := NewTelegramService(newFakeUserRepo(unlinked), newFakeLinkCodes(), bot, "Bot", fixedClock)
		if err := svc.Unlink(ctx, unlinked.ID); err != nil || len(bot.sent) != 0 {
			t.Errorf("Unlink() = %v, mensajes %d", err, len(bot.sent))
		}
	})
}

func TestParseCommand(t *testing.T) {
	tests := []struct{ text, command, arg string }{
		{"/start abc", "/start", "abc"},
		{"  /START   abc  extra", "/start", "abc"},
		{"/start@ByteBalanceBot abc", "/start", "abc"},
		{"/stop", "/stop", ""},
		{"hola", "", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		if command, arg := parseCommand(tt.text); command != tt.command || arg != tt.arg {
			t.Errorf("parseCommand(%q) = %q, %q; se esperaba %q, %q", tt.text, command, arg, tt.command, tt.arg)
		}
	}
}

// isLinked indica si el usuario guardado en el fake tiene Telegram vinculado.
func isLinked(users *fakeUserRepo, id domain.ID) bool {
	u := users.byID[id]
	return u.HasTelegram()
}
