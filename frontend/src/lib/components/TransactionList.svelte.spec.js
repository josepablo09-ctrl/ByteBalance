import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { formatMoney } from '#lib/money.js';
import TransactionList from './TransactionList.svelte';

/**
 * Monto formateado con espacios normales: toHaveTextContent normaliza los del
 * texto (incluido el NBSP de los montos) pero no los del valor esperado.
 *
 * @param {Parameters<typeof formatMoney>} args
 */
const money = (...args) => formatMoney(...args).replace(/\s+/g, ' ');

/** @type {import('#lib/api/finance.js').Transaction} */
const TRANSACTION = {
	id: 't1',
	type: 'expense',
	amount_cents: 4500,
	currency: 'USD',
	category_id: null,
	description: 'Café',
	occurred_at: '2026-10-15T15:00:00Z',
	created_at: ''
};

describe('TransactionList', () => {
	it('ofrece editar y eliminar cada transacción', async () => {
		const onedit = vi.fn();
		const ondelete = vi.fn();
		render(TransactionList, {
			transactions: [TRANSACTION],
			categoriesById: new Map(),
			timeZone: 'UTC',
			onedit,
			ondelete
		});

		await page.getByRole('button', { name: 'Editar Café' }).click();
		await page.getByRole('button', { name: 'Eliminar Café' }).click();

		expect(onedit).toHaveBeenCalledWith(TRANSACTION);
		expect(ondelete).toHaveBeenCalledWith(TRANSACTION);
	});

	it('muestra los movimientos de ahorro según su efecto en el disponible', async () => {
		render(TransactionList, {
			transactions: [
				{ ...TRANSACTION, id: 's1', type: 'saving', description: '', amount_cents: 10000 },
				{ ...TRANSACTION, id: 'w1', type: 'withdrawal', description: 'Viaje', amount_cents: 2500 }
			],
			categoriesById: new Map(),
			timeZone: 'UTC',
			onedit: vi.fn(),
			ondelete: vi.fn()
		});

		// Apartar resta del disponible; retirar del ahorro suma.
		const items = page.getByRole('listitem');
		await expect.element(items.nth(0)).toHaveTextContent('Aporte al ahorro');
		await expect
			.element(items.nth(0))
			.toHaveTextContent(money(-10000, 'USD', { signDisplay: 'always' }));
		await expect.element(items.nth(1)).toHaveTextContent('Viaje');
		await expect.element(items.nth(1)).toHaveTextContent('Retiro del ahorro');
		await expect
			.element(items.nth(1))
			.toHaveTextContent(money(2500, 'USD', { signDisplay: 'always' }));
	});
});
