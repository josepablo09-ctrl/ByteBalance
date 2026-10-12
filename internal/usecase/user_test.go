package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/user"
)

// fakeUserRepo es un user.Repository en memoria. Devuelve copias para que los
// cambios del caso de uso solo se vean si llama a Update.
type fakeUserRepo struct {
	byID      map[domain.ID]user.User
	createErr error
	updates   int
}

var _ user.Repository = (*fakeUserRepo)(nil)

func newFakeUserRepo(users ...user.User) *fakeUserRepo {
	f := &fakeUserRepo{byID: map[domain.ID]user.User{}}
	for _, u := range users {
		f.byID[u.ID] = u
	}
	return f
}

func (f *fakeUserRepo) Create(_ context.Context, u *user.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	for _, existing := range f.byID {
		if strings.EqualFold(existing.Email, u.Email) {
			return domain.ErrConflict
		}
	}
	u.ID = domain.NewID()
	f.byID[u.ID] = *u
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, id domain.ID) (*user.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range f.byID {
		if strings.EqualFold(u.Email, email) {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) GetByTelegramChatID(_ context.Context, chatID int64) (*user.User, error) {
	for _, u := range f.byID {
		if u.TelegramChatID != nil && *u.TelegramChatID == chatID {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) Update(_ context.Context, u *user.User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return domain.ErrNotFound
	}
	f.updates++
	f.byID[u.ID] = *u
	return nil
}

func (f *fakeUserRepo) Delete(_ context.Context, id domain.ID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

// stubHasher simula el hash de contraseñas y cuenta sus llamadas.
type stubHasher struct {
	err   error
	calls int
}

func (h *stubHasher) Hash(password string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + password, nil
}

// existingUser devuelve un usuario persistido válido para sembrar los fakes.
func existingUser(t *testing.T, email, timezone string) user.User {
	t.Helper()
	u, err := user.New(user.Params{
		Email:           email,
		PasswordHash:    "hash",
		FullName:        "Ana Pérez",
		DefaultCurrency: "CRC",
		Timezone:        timezone,
	})
	if err != nil {
		t.Fatalf("user.New: %v", err)
	}
	u.ID = domain.NewID()
	return *u
}

func TestUserServiceRegister(t *testing.T) {
	errDB := errors.New("base caída")
	errHash := errors.New("sin entropía")
	valid := RegisterInput{Email: "Nuevo@Example.com", Password: "contraseña-segura", FullName: "Nuevo Usuario"}

	tests := []struct {
		name       string
		modify     func(in *RegisterInput)
		hasherErr  error
		createErr  error
		wantErr    error
		wantHashed bool // si debe llegar a hashear la contraseña
		check      func(t *testing.T, u *user.User)
	}{
		{
			name:       "aplica valores por defecto",
			modify:     func(*RegisterInput) {},
			wantHashed: true,
			check: func(t *testing.T, u *user.User) {
				if u.ID.IsZero() || u.Email != "nuevo@example.com" || u.DefaultCurrency != "USD" || u.Timezone != "UTC" {
					t.Errorf("usuario registrado = %+v", u)
				}
				if u.PasswordHash != "hashed:contraseña-segura" {
					t.Errorf("PasswordHash = %q, se esperaba el hash y no la contraseña", u.PasswordHash)
				}
			},
		},
		{
			name:       "normaliza la moneda",
			modify:     func(in *RegisterInput) { in.DefaultCurrency = " crc " },
			wantHashed: true,
			check: func(t *testing.T, u *user.User) {
				if u.DefaultCurrency != "CRC" {
					t.Errorf("DefaultCurrency = %q, se esperaba CRC", u.DefaultCurrency)
				}
			},
		},
		{name: "contraseña corta", modify: func(in *RegisterInput) { in.Password = "corta" }, wantErr: domain.ErrInvalidInput},
		{name: "contraseña demasiado larga", modify: func(in *RegisterInput) { in.Password = strings.Repeat("x", 129) }, wantErr: domain.ErrInvalidInput},
		{name: "moneda inválida", modify: func(in *RegisterInput) { in.DefaultCurrency = "dólar" }, wantErr: domain.ErrInvalidInput},
		{name: "email inválido", modify: func(in *RegisterInput) { in.Email = "sin-arroba" }, wantErr: domain.ErrInvalidInput, wantHashed: true},
		{name: "error del hasher", modify: func(*RegisterInput) {}, hasherErr: errHash, wantErr: errHash, wantHashed: true},
		{name: "email ya registrado", modify: func(in *RegisterInput) { in.Email = "ANA@example.com" }, wantErr: ErrEmailTaken, wantHashed: true},
		{name: "error de la base", modify: func(*RegisterInput) {}, createErr: errDB, wantErr: errDB, wantHashed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeUserRepo(existingUser(t, "ana@example.com", "UTC"))
			repo.createErr = tt.createErr
			hasher := &stubHasher{err: tt.hasherErr}
			svc := NewUserService(repo, hasher)

			in := valid
			tt.modify(&in)
			u, err := svc.Register(context.Background(), in)

			if (hasher.calls > 0) != tt.wantHashed {
				t.Errorf("llamadas al hasher = %d, wantHashed %v", hasher.calls, tt.wantHashed)
			}
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Register() = %+v, se esperaba error", u)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Register() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if len(repo.byID) != 1 {
					t.Errorf("se persistieron %d usuarios, se esperaba solo el existente", len(repo.byID))
				}
				return
			}
			if err != nil {
				t.Fatalf("Register() error inesperado: %v", err)
			}
			if _, ok := repo.byID[u.ID]; !ok {
				t.Error("el usuario no quedó persistido")
			}
			tt.check(t, u)
		})
	}
}

func TestEmailTakenIsConflict(t *testing.T) {
	if !errors.Is(ErrEmailTaken, domain.ErrConflict) {
		t.Fatal("ErrEmailTaken debe envolver domain.ErrConflict")
	}
}

func TestUserServiceUpdatePreferences(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name    string
		userID  func(existing domain.ID) domain.ID
		in      PreferencesInput
		wantErr error
		want    func(t *testing.T, u user.User)
	}{
		{
			name: "cambia todos los campos",
			in:   PreferencesInput{FullName: ptr(" Ana María "), DefaultCurrency: ptr("eur"), Timezone: ptr("Europe/Madrid")},
			want: func(t *testing.T, u user.User) {
				if u.FullName != "Ana María" || u.DefaultCurrency != "EUR" || u.Timezone != "Europe/Madrid" {
					t.Errorf("usuario actualizado = %+v", u)
				}
			},
		},
		{
			name: "los campos nil no cambian",
			in:   PreferencesInput{Timezone: ptr("America/Bogota")},
			want: func(t *testing.T, u user.User) {
				if u.FullName != "Ana Pérez" || u.DefaultCurrency != "CRC" || u.Timezone != "America/Bogota" {
					t.Errorf("usuario actualizado = %+v", u)
				}
			},
		},
		{name: "moneda inválida", in: PreferencesInput{DefaultCurrency: ptr("XX")}, wantErr: domain.ErrInvalidInput},
		{name: "zona horaria desconocida", in: PreferencesInput{Timezone: ptr("Marte/Olympus")}, wantErr: domain.ErrInvalidInput},
		{name: "zona horaria vacía", in: PreferencesInput{Timezone: ptr("  ")}, wantErr: domain.ErrInvalidInput},
		{name: "nombre vacío", in: PreferencesInput{FullName: ptr("")}, wantErr: domain.ErrInvalidInput},
		{
			name:    "usuario inexistente",
			userID:  func(domain.ID) domain.ID { return domain.NewID() },
			in:      PreferencesInput{FullName: ptr("Otro")},
			wantErr: domain.ErrNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := existingUser(t, "ana@example.com", "America/Costa_Rica")
			repo := newFakeUserRepo(existing)
			svc := NewUserService(repo, &stubHasher{})

			id := existing.ID
			if tt.userID != nil {
				id = tt.userID(existing.ID)
			}
			_, err := svc.UpdatePreferences(context.Background(), id, tt.in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UpdatePreferences() error = %v, se esperaba %v", err, tt.wantErr)
				}
				if repo.updates != 0 || repo.byID[existing.ID] != existing {
					t.Error("un error de validación no debe guardar cambios")
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdatePreferences() error inesperado: %v", err)
			}
			if repo.updates != 1 {
				t.Errorf("Update llamado %d veces, se esperaba 1", repo.updates)
			}
			tt.want(t, repo.byID[existing.ID])
		})
	}
}
