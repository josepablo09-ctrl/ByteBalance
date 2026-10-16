// Cálculos de hábitos sobre fechas de calendario `YYYY-MM-DD`. Los días ya vienen
// en la zona horaria del usuario (logged_on en el backend; "hoy" con todayInZone),
// así que aquí la aritmética es de calendario puro, en UTC, sin horas.

/** @typedef {import('./api/habits.js').Frequency} Frequency */
/** @typedef {import('./api/habits.js').Habit} Habit */
/** @typedef {import('./api/habits.js').HabitLog} HabitLog */

// Límites del formulario. El backend solo exige nombre ≤ 100 y meta > 0; el
// resto son topes de interfaz para mantener los datos razonables.
export const MAX_NAME_LENGTH = 100;
export const MAX_DESCRIPTION_LENGTH = 500;
export const MAX_NOTE_LENGTH = 280;
export const MAX_TARGET = 1000;
export const MAX_COUNT = 1000;

/** Días visibles en la tira de historial de cada hábito. */
export const HISTORY_DAYS = 14;

export const FREQUENCY_OPTIONS = [
	{ value: 'daily', label: 'Diario' },
	{ value: 'weekly', label: 'Semanal' },
	{ value: 'monthly', label: 'Mensual' }
];

/** @type {Record<Frequency, string>} */
const PERIOD_NAMES = { daily: 'hoy', weekly: 'esta semana', monthly: 'este mes' };

/** @param {string} date `YYYY-MM-DD` */
function toUtc(date) {
	const [y, m, d] = date.split('-').map(Number);
	return Date.UTC(y, m - 1, d);
}

/** @param {number} ms */
function fromUtc(ms) {
	return new Date(ms).toISOString().slice(0, 10);
}

/**
 * @param {string} date `YYYY-MM-DD`
 * @param {number} days Puede ser negativo.
 */
export function addDays(date, days) {
	return fromUtc(toUtc(date) + days * 86_400_000);
}

/** Lunes de la semana de `date` (semana ISO, de lunes a domingo). */
export function weekStart(/** @type {string} */ date) {
	const weekday = new Date(toUtc(date)).getUTCDay(); // 0 = domingo
	return addDays(date, -((weekday + 6) % 7));
}

/** Primer día del mes de `date`. */
export function monthStart(/** @type {string} */ date) {
	return `${date.slice(0, 8)}01`;
}

/**
 * Inicio del periodo de la meta que contiene `date`.
 *
 * @param {Frequency} frequency
 * @param {string} date
 */
export function periodStart(frequency, date) {
	if (frequency === 'weekly') return weekStart(date);
	if (frequency === 'monthly') return monthStart(date);
	return date;
}

/**
 * Rango de registros que necesita la pantalla: cubre la tira de historial y el
 * periodo actual de cualquier frecuencia (semana o mes), hasta hoy.
 *
 * @param {string} today
 * @returns {{ from: string, to: string }}
 */
export function logsRange(today) {
	const candidates = [addDays(today, -(HISTORY_DAYS - 1)), weekStart(today), monthStart(today)];
	return { from: candidates.sort()[0], to: today };
}

/**
 * Progreso de la meta en el periodo actual.
 *
 * @param {Habit} habit
 * @param {HabitLog[]} logs
 * @param {string} today
 */
export function periodProgress(habit, logs, today) {
	const start = periodStart(habit.frequency, today);
	const done = logs
		.filter((l) => l.date >= start && l.date <= today)
		.reduce((sum, l) => sum + l.count, 0);
	return {
		done,
		target: habit.target_count,
		isComplete: done >= habit.target_count,
		periodName: PERIOD_NAMES[habit.frequency]
	};
}

/**
 * Cantidad registrada en un día.
 *
 * @param {HabitLog[]} logs
 * @param {string} date
 */
export function countOn(logs, date) {
	return logs.find((l) => l.date === date)?.count ?? 0;
}

/**
 * Últimos `days` días hasta hoy, del más antiguo al más reciente.
 *
 * @param {HabitLog[]} logs
 * @param {string} today
 * @param {number} [days]
 * @returns {{ date: string, count: number }[]}
 */
export function recentDays(logs, today, days = HISTORY_DAYS) {
	return Array.from({ length: days }, (_, i) => {
		const date = addDays(today, i - (days - 1));
		return { date, count: countOn(logs, date) };
	});
}

/**
 * Racha de días consecutivos en que se cumplió la meta de un hábito diario. Si
 * hoy aún no se cumple, la racha cuenta hasta ayer (todavía se puede salvar).
 * Solo considera los registros recibidos, así que su máximo es el rango cargado.
 *
 * @param {Habit} habit
 * @param {HabitLog[]} logs
 * @param {string} today
 */
export function dailyStreak(habit, logs, today) {
	if (habit.frequency !== 'daily') return 0;
	const met = (/** @type {string} */ date) => countOn(logs, date) >= habit.target_count;
	let day = met(today) ? today : addDays(today, -1);
	let streak = 0;
	while (met(day)) {
		streak++;
		day = addDays(day, -1);
	}
	return streak;
}

/**
 * Describe la meta, p. ej. "1 vez al día" o "3 veces por semana".
 *
 * @param {Pick<Habit, 'frequency' | 'target_count'>} habit
 */
export function describeGoal({ frequency, target_count }) {
	const times = target_count === 1 ? '1 vez' : `${target_count} veces`;
	const per = { daily: 'al día', weekly: 'por semana', monthly: 'al mes' }[frequency];
	return `${times} ${per}`;
}

/**
 * @typedef {object} HabitDraft Valores del formulario de hábito.
 * @property {string} name
 * @property {string} description
 * @property {Frequency} frequency
 * @property {string} target Meta tal como la escribió el usuario.
 * @property {boolean} hasReminder
 * @property {string} reminderTime `HH:MM`
 */

/** @param {string} value @param {number} max */
function parseCount(value, max) {
	if (!/^\d+$/.test(value.trim())) return undefined;
	const n = Number(value);
	return n >= 1 && n <= max ? n : undefined;
}

/**
 * Valida el formulario y construye el cuerpo para crear o editar el hábito.
 * Al editar se envían todos los campos; el recordatorio desactivado va como
 * `null` para quitarlo.
 *
 * @param {HabitDraft} draft
 * @returns {{ input: import('./api/habits.js').HabitInput, errors?: undefined }
 *   | { input?: undefined, errors: Partial<Record<keyof HabitDraft, string>> }}
 */
export function buildHabitInput(draft) {
	/** @type {Partial<Record<keyof HabitDraft, string>>} */
	const errors = {};
	const name = draft.name.trim();
	const description = draft.description.trim();
	const target = parseCount(draft.target, MAX_TARGET);

	if (!name) errors.name = 'Ponle un nombre.';
	else if ([...name].length > MAX_NAME_LENGTH)
		errors.name = `Usa como máximo ${MAX_NAME_LENGTH} caracteres.`;
	if ([...description].length > MAX_DESCRIPTION_LENGTH) {
		errors.description = `Usa como máximo ${MAX_DESCRIPTION_LENGTH} caracteres.`;
	}
	if (target === undefined) errors.target = `Usa un número entero entre 1 y ${MAX_TARGET}.`;
	if (draft.hasReminder && !/^([01]\d|2[0-3]):[0-5]\d$/.test(draft.reminderTime)) {
		errors.reminderTime = 'Elige una hora.';
	}

	if (Object.keys(errors).length > 0 || target === undefined) return { errors };
	return {
		input: {
			name,
			description,
			frequency: draft.frequency,
			target_count: target,
			reminder_time: draft.hasReminder ? draft.reminderTime : null
		}
	};
}

/**
 * Valida el formulario de registro de un día.
 *
 * @param {{ date: string, count: string, note: string }} draft
 * @param {string} today
 * @returns {{ input: import('./api/habits.js').LogInput, errors?: undefined }
 *   | { input?: undefined, errors: Partial<Record<'date' | 'count' | 'note', string>> }}
 */
export function buildLogInput(draft, today) {
	/** @type {Partial<Record<'date' | 'count' | 'note', string>>} */
	const errors = {};
	const count = parseCount(draft.count, MAX_COUNT);
	const note = draft.note.trim();

	if (!/^\d{4}-\d{2}-\d{2}$/.test(draft.date)) errors.date = 'Elige una fecha.';
	else if (draft.date > today) errors.date = 'No puedes registrar un día futuro.';
	if (count === undefined) errors.count = `Usa un número entero entre 1 y ${MAX_COUNT}.`;
	if ([...note].length > MAX_NOTE_LENGTH) {
		errors.note = `Usa como máximo ${MAX_NOTE_LENGTH} caracteres.`;
	}

	if (Object.keys(errors).length > 0 || count === undefined) return { errors };
	return { input: { date: draft.date, count, note } };
}
