import { describe, expect, it } from 'vitest';
import {
	addDays,
	buildHabitInput,
	buildLogInput,
	dailyStreak,
	describeGoal,
	logsRange,
	monthStart,
	periodProgress,
	recentDays,
	weekStart
} from './habits.js';

/**
 * @param {Partial<import('./api/habits.js').Habit>} overrides
 * @returns {import('./api/habits.js').Habit}
 */
const habit = (overrides) => ({
	id: 'h1',
	name: 'Leer',
	description: '',
	frequency: 'daily',
	target_count: 1,
	reminder_time: null,
	is_active: true,
	created_at: '',
	...overrides
});

/** @param {Record<string, number>} counts */
const logs = (counts) =>
	Object.entries(counts).map(([date, count]) => ({
		id: date,
		habit_id: 'h1',
		date,
		count,
		note: ''
	}));

// Viernes 16 de octubre de 2026.
const TODAY = '2026-10-16';

describe('aritmética de calendario', () => {
	it.each([
		['2026-10-16', 1, '2026-10-17'],
		['2026-10-31', 1, '2026-11-01'],
		['2026-03-01', -1, '2026-02-28'],
		['2028-03-01', -1, '2028-02-29'], // bisiesto
		['2026-01-01', -1, '2025-12-31']
	])('addDays(%s, %d) = %s', (date, days, want) => {
		expect(addDays(date, days)).toBe(want);
	});

	it.each([
		['2026-10-16', '2026-10-12'], // viernes → lunes
		['2026-10-12', '2026-10-12'], // lunes
		['2026-10-18', '2026-10-12'], // domingo pertenece a la semana del lunes anterior
		['2026-11-01', '2026-10-26'] // la semana cruza el mes
	])('weekStart(%s) = %s', (date, want) => {
		expect(weekStart(date)).toBe(want);
	});

	it('monthStart', () => {
		expect(monthStart('2026-10-16')).toBe('2026-10-01');
	});

	it('logsRange cubre la tira de 14 días, la semana y el mes', () => {
		expect(logsRange(TODAY)).toEqual({ from: '2026-10-01', to: TODAY });
		// A principios de mes manda la tira de 14 días.
		expect(logsRange('2026-10-02')).toEqual({ from: '2026-09-19', to: '2026-10-02' });
	});
});

describe('periodProgress', () => {
	const history = logs({ '2026-09-30': 5, '2026-10-05': 1, '2026-10-12': 2, '2026-10-16': 1 });

	it.each([
		['daily', 1, 1, true, 'hoy'],
		['weekly', 3, 3, true, 'esta semana'], // 12 y 16 de octubre
		['monthly', 10, 4, false, 'este mes'] // 5, 12 y 16 de octubre; septiembre no cuenta
	])('%s', (frequency, target, done, isComplete, periodName) => {
		const h = habit({ frequency: /** @type {any} */ (frequency), target_count: target });
		expect(periodProgress(h, history, TODAY)).toEqual({ done, target, isComplete, periodName });
	});
});

describe('dailyStreak', () => {
	it('cuenta días consecutivos que cumplen la meta hasta hoy', () => {
		const h = habit({ target_count: 2 });
		const l = logs({ '2026-10-13': 2, '2026-10-14': 3, '2026-10-15': 2, '2026-10-16': 2 });
		expect(dailyStreak(h, l, TODAY)).toBe(4);
	});

	it('si hoy aún no se cumple, cuenta hasta ayer', () => {
		const l = logs({ '2026-10-14': 1, '2026-10-15': 1 });
		expect(dailyStreak(habit({}), l, TODAY)).toBe(2);
	});

	it('un día por debajo de la meta corta la racha', () => {
		const h = habit({ target_count: 2 });
		const l = logs({ '2026-10-14': 2, '2026-10-15': 1, '2026-10-16': 2 });
		expect(dailyStreak(h, l, TODAY)).toBe(1);
	});

	it('no aplica a hábitos semanales o mensuales', () => {
		expect(dailyStreak(habit({ frequency: 'weekly' }), logs({ [TODAY]: 9 }), TODAY)).toBe(0);
	});
});

describe('recentDays', () => {
	it('devuelve los últimos días en orden, con ceros', () => {
		expect(recentDays(logs({ '2026-10-15': 2 }), TODAY, 3)).toEqual([
			{ date: '2026-10-14', count: 0 },
			{ date: '2026-10-15', count: 2 },
			{ date: '2026-10-16', count: 0 }
		]);
	});
});

describe('describeGoal', () => {
	it.each([
		['daily', 1, '1 vez al día'],
		['weekly', 3, '3 veces por semana'],
		['monthly', 2, '2 veces al mes']
	])('%s × %d', (frequency, target_count, want) => {
		expect(describeGoal({ frequency: /** @type {any} */ (frequency), target_count })).toBe(want);
	});
});

describe('buildHabitInput', () => {
	/** @type {import('./habits.js').HabitDraft} */
	const draft = {
		name: '  Leer  ',
		description: ' 20 páginas ',
		frequency: 'weekly',
		target: '3',
		hasReminder: true,
		reminderTime: '21:30'
	};

	it('limpia los textos y convierte la meta', () => {
		expect(buildHabitInput(draft)).toEqual({
			input: {
				name: 'Leer',
				description: '20 páginas',
				frequency: 'weekly',
				target_count: 3,
				reminder_time: '21:30'
			}
		});
	});

	it('sin recordatorio envía null para quitarlo al editar', () => {
		expect(buildHabitInput({ ...draft, hasReminder: false }).input?.reminder_time).toBeNull();
	});

	it.each([
		['name', { name: '  ' }],
		['name', { name: 'x'.repeat(101) }],
		['target', { target: '0' }],
		['target', { target: '2.5' }],
		['target', { target: '1001' }],
		['reminderTime', { reminderTime: '' }],
		['description', { description: 'x'.repeat(501) }]
	])('marca %s como inválido', (field, override) => {
		expect(Object.keys(buildHabitInput({ ...draft, ...override }).errors ?? {})).toEqual([field]);
	});
});

describe('buildLogInput', () => {
	it('acepta un día pasado', () => {
		expect(buildLogInput({ date: '2026-10-10', count: '2', note: ' bien ' }, TODAY)).toEqual({
			input: { date: '2026-10-10', count: 2, note: 'bien' }
		});
	});

	it('rechaza días futuros y cantidades inválidas', () => {
		const { errors } = buildLogInput({ date: '2026-10-17', count: '0', note: '' }, TODAY);
		expect(Object.keys(errors ?? {}).sort()).toEqual(['count', 'date']);
	});
});
