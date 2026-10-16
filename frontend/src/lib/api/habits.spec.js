import { describe, expect, it, vi } from 'vitest';
import { createHabitsApi } from './habits.js';

/** Cliente simulado que registra las llamadas y responde `response`. */
function fakeClient(/** @type {unknown} */ response) {
	return {
		request: vi.fn(),
		get: vi.fn(async () => response),
		post: vi.fn(async () => response),
		patch: vi.fn(async () => response),
		delete: vi.fn(async () => null)
	};
}

describe('createHabitsApi', () => {
	it('lista todos los hábitos por defecto y solo activos si se pide', async () => {
		const client = fakeClient({ habits: [{ id: 'h1' }] });
		const api = createHabitsApi(client);

		await expect(api.listHabits()).resolves.toEqual([{ id: 'h1' }]);
		await api.listHabits({ activeOnly: true });

		expect(client.get).toHaveBeenNthCalledWith(1, '/habits', { query: { active: undefined } });
		expect(client.get).toHaveBeenNthCalledWith(2, '/habits', { query: { active: true } });
	});

	it('pide los registros de un rango de fechas', async () => {
		const client = fakeClient({ logs: [] });
		await createHabitsApi(client).listLogs('h1', { from: '2026-10-01', to: '2026-10-16' });
		expect(client.get).toHaveBeenCalledWith('/habits/h1/logs', {
			query: { from: '2026-10-01', to: '2026-10-16' }
		});
	});

	it('pausa con PATCH y elimina con DELETE', async () => {
		const client = fakeClient({ id: 'h1' });
		const api = createHabitsApi(client);
		await api.updateHabit('h1', { is_active: false });
		await api.deleteHabit('h1');
		expect(client.patch).toHaveBeenCalledWith('/habits/h1', { is_active: false });
		expect(client.delete).toHaveBeenCalledWith('/habits/h1');
	});

	it('registra el cumplimiento de hoy sin cuerpo adicional', async () => {
		const client = fakeClient({ id: 'l1' });
		await createHabitsApi(client).logCompletion('h1');
		expect(client.post).toHaveBeenCalledWith('/habits/h1/logs', {});
	});

	it('borra el registro de un día con DELETE', async () => {
		const client = fakeClient(null);
		await createHabitsApi(client).deleteLog('h1', '2026-10-16');
		expect(client.delete).toHaveBeenCalledWith('/habits/h1/logs/2026-10-16');
	});

	it('codifica el id en la ruta', async () => {
		const client = fakeClient({ logs: [] });
		await createHabitsApi(client).listLogs('a/b');
		expect(client.get).toHaveBeenCalledWith('/habits/a%2Fb/logs', expect.anything());
	});
});
