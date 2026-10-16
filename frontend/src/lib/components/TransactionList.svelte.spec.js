import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import TransactionList from './TransactionList.svelte';

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
});
