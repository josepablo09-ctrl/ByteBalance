import { describe, expect, it, vi } from 'vitest';
import { createFinanceApi } from './finance.js';

/** Cliente simulado que registra las llamadas y responde `response`. */
function fakeClient(/** @type {unknown} */ response) {
	return {
		request: vi.fn(),
		get: vi.fn(async () => response),
		post: vi.fn(async () => response),
		patch: vi.fn(),
		delete: vi.fn()
	};
}

describe('createFinanceApi', () => {
	it('lista transacciones con los filtros como parámetros de consulta', async () => {
		const client = fakeClient({ transactions: [{ id: 't1' }] });
		const api = createFinanceApi(client);

		const result = await api.listTransactions({
			from: new Date('2026-10-01T06:00:00Z'),
			to: new Date('2026-11-01T06:00:00Z'),
			type: 'expense',
			limit: 20,
			offset: 40
		});

		expect(result).toEqual([{ id: 't1' }]);
		expect(client.get).toHaveBeenCalledWith('/transactions', {
			query: {
				from: '2026-10-01T06:00:00.000Z',
				to: '2026-11-01T06:00:00.000Z',
				type: 'expense',
				category_id: undefined,
				limit: 20,
				offset: 40
			}
		});
	});

	it('pide el balance histórico sin fechas', async () => {
		const client = fakeClient({ balances: [] });
		await createFinanceApi(client).balance();
		expect(client.get).toHaveBeenCalledWith('/balance', {
			query: { from: undefined, to: undefined }
		});
	});

	it('desempaqueta las categorías', async () => {
		const client = fakeClient({ categories: [{ id: 'c1' }] });
		await expect(createFinanceApi(client).listCategories()).resolves.toEqual([{ id: 'c1' }]);
	});

	it('crea categorías y transacciones con POST', async () => {
		const client = fakeClient({ id: 'x' });
		const api = createFinanceApi(client);
		await api.createCategory('Salario', 'income');
		await api.recordTransaction({ type: 'income', amount_cents: 100 });
		expect(client.post).toHaveBeenNthCalledWith(1, '/categories', {
			name: 'Salario',
			type: 'income'
		});
		expect(client.post).toHaveBeenNthCalledWith(2, '/transactions', {
			type: 'income',
			amount_cents: 100
		});
	});
});

describe('edición y borrado de transacciones', () => {
	it('corrige con PATCH y elimina con DELETE', async () => {
		const client = {
			request: vi.fn(),
			get: vi.fn(),
			post: vi.fn(),
			patch: vi.fn(async () => ({ id: 't1' })),
			delete: vi.fn(async () => null)
		};
		const api = createFinanceApi(client);
		await api.updateTransaction('t1', { amount_cents: 500, category_id: null });
		await api.deleteTransaction('t1');
		expect(client.patch).toHaveBeenCalledWith('/transactions/t1', {
			amount_cents: 500,
			category_id: null
		});
		expect(client.delete).toHaveBeenCalledWith('/transactions/t1');
	});
});
