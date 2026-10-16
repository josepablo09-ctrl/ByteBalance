import { describe, expect, it } from 'vitest';
import { buildTransactionInput, validateCategoryName } from './transactions.js';

const context = { timeZone: 'America/Costa_Rica', today: '2026-10-15' };

/** @type {import('./transactions.js').TransactionDraft} */
const draft = {
	type: 'expense',
	amount: '1250.50',
	currency: 'CRC',
	categoryId: '',
	description: '  Supermercado  ',
	date: '2026-10-15'
};

describe('buildTransactionInput', () => {
	it('una transacción de hoy se envía sin fecha para que el backend use la hora actual', () => {
		expect(buildTransactionInput(draft, context)).toEqual({
			input: {
				type: 'expense',
				amount_cents: 125050,
				currency: 'CRC',
				category_id: null,
				description: 'Supermercado'
			}
		});
	});

	it('fecha un día pasado a las 12:00 en la zona del usuario', () => {
		const { input } = buildTransactionInput({ ...draft, date: '2026-10-01' }, context);
		// 12:00 en Costa Rica (UTC-6) = 18:00 UTC.
		expect(input?.occurred_at).toBe('2026-10-01T18:00:00.000Z');
	});

	it('incluye la categoría elegida', () => {
		const { input } = buildTransactionInput({ ...draft, categoryId: 'cat-1' }, context);
		expect(input?.category_id).toBe('cat-1');
	});

	it('reporta todos los errores a la vez', () => {
		const { errors } = buildTransactionInput(
			{ ...draft, amount: '0', currency: '', description: 'x'.repeat(256), date: '' },
			context
		);
		expect(Object.keys(errors ?? {}).sort()).toEqual(['amount', 'currency', 'date', 'description']);
	});
});

describe('validateCategoryName', () => {
	it.each([
		['Supermercado', undefined],
		['   ', 'Escribe un nombre.'],
		['x'.repeat(81), 'Usa como máximo 80 caracteres.']
	])('%j', (name, want) => {
		expect(validateCategoryName(name)).toBe(want);
	});
});

describe('buildTransactionInput al editar', () => {
	const editing = { ...context, originalDate: '2026-10-05', now: new Date('2026-10-15T20:15:00Z') };

	it('no envía la fecha si no cambió, para conservar la hora original', () => {
		const { input } = buildTransactionInput({ ...draft, date: '2026-10-05' }, editing);
		expect(input).not.toHaveProperty('occurred_at');
	});

	it('mover a hoy usa la hora actual, no las 12:00 (que podría estar en el futuro)', () => {
		const { input } = buildTransactionInput({ ...draft, date: '2026-10-15' }, editing);
		expect(input?.occurred_at).toBe('2026-10-15T20:15:00.000Z');
	});

	it('mover a otro día lo fecha a las 12:00 del usuario', () => {
		const { input } = buildTransactionInput({ ...draft, date: '2026-10-01' }, editing);
		expect(input?.occurred_at).toBe('2026-10-01T18:00:00.000Z');
	});

	it('sin categoría envía null para quitarla', () => {
		const { input } = buildTransactionInput({ ...draft, date: '2026-10-05' }, editing);
		expect(input?.category_id).toBeNull();
	});
});
