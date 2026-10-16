package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bytebalance/internal/domain"
	"bytebalance/internal/domain/habit"
)

// Rango de fechas (en días, ambos extremos incluidos) al listar registros.
const (
	defaultLogRangeDays = 30
	maxLogRangeDays     = 366
)

// HabitService implementa los casos de uso de hábitos: creación, consulta y
// registro del cumplimiento diario.
type HabitService struct {
	habits habit.Repository
	logs   habit.LogRepository
	users  userFinder
	now    func() time.Time
}

// NewHabitService crea un HabitService. Si now es nil se usa time.Now.
func NewHabitService(habits habit.Repository, logs habit.LogRepository, users userFinder, now func() time.Time) *HabitService {
	return &HabitService{habits: habits, logs: logs, users: users, now: clockOrDefault(now)}
}

// CreateHabitInput son los datos de un hábito nuevo.
type CreateHabitInput struct {
	Name        string
	Description string
	// Frequency vacía equivale a diaria.
	Frequency habit.Frequency
	// TargetCount cero equivale a 1.
	TargetCount int
	// ReminderTime es opcional y se interpreta en la zona horaria del usuario.
	ReminderTime *habit.TimeOfDay
}

// CreateHabit crea un hábito activo para el usuario.
func (s *HabitService) CreateHabit(ctx context.Context, userID domain.ID, in CreateHabitInput) (*habit.Habit, error) {
	h, err := habit.New(habit.Params{
		UserID:       userID,
		Name:         in.Name,
		Description:  in.Description,
		Frequency:    in.Frequency,
		TargetCount:  in.TargetCount,
		ReminderTime: in.ReminderTime,
	})
	if err != nil {
		return nil, fmt.Errorf("crear hábito: %w", err)
	}
	if err := s.habits.Create(ctx, h); err != nil {
		return nil, fmt.Errorf("crear hábito: %w", err)
	}
	return h, nil
}

// UpdateHabitInput son los cambios a un hábito. Un campo nil se deja igual.
type UpdateHabitInput struct {
	Name        *string
	Description *string
	Frequency   *habit.Frequency
	TargetCount *int
	// SetReminder indica si se modifica el recordatorio; entonces ReminderTime
	// nil lo quita.
	SetReminder  bool
	ReminderTime *habit.TimeOfDay
	// IsActive pausa (false) o reanuda (true) el hábito.
	IsActive *bool
}

// UpdateHabit modifica un hábito del usuario. Si algún valor es inválido no se
// guarda ningún cambio. Un hábito ajeno se reporta como inexistente.
func (s *HabitService) UpdateHabit(ctx context.Context, userID, habitID domain.ID, in UpdateHabitInput) (*habit.Habit, error) {
	h, err := s.habits.GetByID(ctx, userID, habitID)
	if err != nil {
		return nil, fmt.Errorf("actualizar hábito: %w", err)
	}
	if in.Name != nil {
		h.Name = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		h.Description = strings.TrimSpace(*in.Description)
	}
	if in.Frequency != nil {
		h.Frequency = *in.Frequency
	}
	if in.TargetCount != nil {
		h.TargetCount = *in.TargetCount
	}
	if in.SetReminder {
		h.ReminderTime = in.ReminderTime
	}
	if in.IsActive != nil {
		if *in.IsActive {
			h.Activate()
		} else {
			h.Deactivate()
		}
	}
	if err := h.Validate(); err != nil {
		return nil, fmt.Errorf("actualizar hábito: %w", err)
	}

	if err := s.habits.Update(ctx, h); err != nil {
		return nil, fmt.Errorf("actualizar hábito: %w", err)
	}
	return h, nil
}

// DeleteHabit elimina (borrado lógico) un hábito del usuario y deja de
// mostrarlo junto con su historial. Un hábito ajeno se reporta como inexistente.
func (s *HabitService) DeleteHabit(ctx context.Context, userID, habitID domain.ID) error {
	if err := s.habits.Delete(ctx, userID, habitID); err != nil {
		return fmt.Errorf("eliminar hábito: %w", err)
	}
	return nil
}

// LogCompletionInput es el cumplimiento de un hábito en un día.
type LogCompletionInput struct {
	// Date es el día cumplido (solo cuenta la fecha); su valor cero equivale a
	// hoy en la zona horaria del usuario.
	Date  time.Time
	Count int
	Note  string
}

// LogCompletion registra el cumplimiento de un hábito del usuario. Si ya había
// un registro ese día, lo reemplaza. No admite días futuros ni hábitos pausados.
func (s *HabitService) LogCompletion(ctx context.Context, userID, habitID domain.ID, in LogCompletionInput) (*habit.Log, error) {
	// GetByID está acotado al usuario: un hábito ajeno se reporta como inexistente.
	h, err := s.habits.GetByID(ctx, userID, habitID)
	if err != nil {
		return nil, fmt.Errorf("registrar cumplimiento: %w", err)
	}

	today, err := s.userToday(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("registrar cumplimiento: %w", err)
	}
	day := today
	if !in.Date.IsZero() {
		day = habit.DateOf(in.Date)
		if day.After(today) {
			return nil, fmt.Errorf("registrar cumplimiento: %w: no se puede registrar un día futuro", domain.ErrInvalidInput)
		}
	}

	l, err := h.Record(day, in.Count, in.Note)
	if err != nil {
		return nil, fmt.Errorf("registrar cumplimiento: %w", err)
	}
	if err := s.logs.Upsert(ctx, l); err != nil {
		return nil, fmt.Errorf("registrar cumplimiento: %w", err)
	}
	return l, nil
}

// ListHabits devuelve los hábitos del usuario; con activeOnly omite los pausados.
func (s *HabitService) ListHabits(ctx context.Context, userID domain.ID, activeOnly bool) ([]habit.Habit, error) {
	habits, err := s.habits.ListByUser(ctx, userID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("listar hábitos: %w", err)
	}
	return habits, nil
}

// ListLogs devuelve los registros de un hábito del usuario con fecha en
// [from, to] (solo cuenta la fecha), ordenados por fecha. Si to es cero se usa
// hoy en la zona horaria del usuario; si from es cero, los 30 días que terminan
// en to. El rango no puede superar 366 días.
func (s *HabitService) ListLogs(ctx context.Context, userID, habitID domain.ID, from, to time.Time) ([]habit.Log, error) {
	// Comprueba la propiedad: el repositorio de registros no la valida.
	if _, err := s.habits.GetByID(ctx, userID, habitID); err != nil {
		return nil, fmt.Errorf("listar registros: %w", err)
	}

	to = habit.DateOf(to)
	if to.IsZero() {
		today, err := s.userToday(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("listar registros: %w", err)
		}
		to = today
	}
	from = habit.DateOf(from)
	if from.IsZero() {
		from = to.AddDate(0, 0, -(defaultLogRangeDays - 1))
	}
	if from.After(to) {
		return nil, fmt.Errorf("listar registros: %w: la fecha inicial es posterior a la final", domain.ErrInvalidInput)
	}
	if to.Sub(from) >= maxLogRangeDays*24*time.Hour {
		return nil, fmt.Errorf("listar registros: %w: el rango no puede superar %d días", domain.ErrInvalidInput, maxLogRangeDays)
	}

	logs, err := s.logs.ListByHabit(ctx, habitID, from, to)
	if err != nil {
		return nil, fmt.Errorf("listar registros: %w", err)
	}
	return logs, nil
}

// DeleteLog borra el registro de un día de un hábito del usuario, dejando ese
// día en cero (p. ej. para deshacer un "hecho" marcado por error). Se admite
// aunque el hábito esté pausado, para poder corregir el historial. Devuelve
// domain.ErrNotFound si el hábito es ajeno o ese día no tiene registro.
func (s *HabitService) DeleteLog(ctx context.Context, userID, habitID domain.ID, date time.Time) error {
	// Comprueba la propiedad: el repositorio de registros no la valida.
	if _, err := s.habits.GetByID(ctx, userID, habitID); err != nil {
		return fmt.Errorf("borrar registro: %w", err)
	}
	if err := s.logs.Delete(ctx, habitID, habit.DateOf(date)); err != nil {
		return fmt.Errorf("borrar registro: %w", err)
	}
	return nil
}

// userToday devuelve la fecha actual en la zona horaria del usuario.
func (s *HabitService) userToday(ctx context.Context, userID domain.ID) (time.Time, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("obtener usuario: %w", err)
	}
	loc, err := u.Location()
	if err != nil {
		return time.Time{}, err
	}
	return habit.DateOf(s.now().In(loc)), nil
}
