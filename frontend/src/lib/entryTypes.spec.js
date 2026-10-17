import { describe, expect, it } from 'vitest';
import { availableEffect, ENTRY_TYPES, goalProgress } from './entryTypes.js';

describe('availableEffect', () => {
	it.each([
		['income', 1000, 1000],
		['expense', 1000, -1000],
		['saving', 1000, -1000],
		['withdrawal', 1000, 1000]
	])('%s de %i cambia el disponible en %i', (type, amount, want) => {
		const effect = availableEffect({
			type: /** @type {import('./api/finance.js').EntryType} */ (type),
			amount_cents: amount
		});
		expect(effect).toBe(want);
	});
});

describe('ENTRY_TYPES', () => {
	it('solo ingresos y gastos admiten categorías', () => {
		expect(ENTRY_TYPES.income.categorizable).toBe(true);
		expect(ENTRY_TYPES.expense.categorizable).toBe(true);
		expect(ENTRY_TYPES.saving.categorizable).toBe(false);
		expect(ENTRY_TYPES.withdrawal.categorizable).toBe(false);
	});
});

describe('goalProgress', () => {
	it.each([
		[60_000_00, 100_000_00, 60],
		[99_999_99, 100_000_00, 99], // no se muestra 100 % antes de cumplirla
		[100_000_00, 100_000_00, 100],
		[250_000_00, 100_000_00, 100],
		[-5_000_00, 100_000_00, 0],
		[50_000_00, null, null],
		[50_000_00, 0, null]
	])('ahorro %i con meta %s → %s', (saved, goal, want) => {
		expect(goalProgress(saved, goal)).toBe(want);
	});
});
