import { describe, expect, it, vi } from 'vitest';
import { createTelegramApi } from './telegram.js';

describe('createTelegramApi', () => {
	it('usa los endpoints de Telegram', async () => {
		const client = {
			request: vi.fn(),
			get: vi.fn(async () => ({ enabled: true, linked: false })),
			post: vi.fn(async () => ({ code: 'c', url: 'u', expires_at: '' })),
			patch: vi.fn(),
			delete: vi.fn(async () => null)
		};
		const api = createTelegramApi(client);

		await api.status();
		await api.createLink();
		await api.sendTest();
		await api.unlink();

		expect(client.get).toHaveBeenCalledWith('/telegram');
		expect(client.post).toHaveBeenNthCalledWith(1, '/telegram/link');
		expect(client.post).toHaveBeenNthCalledWith(2, '/telegram/test');
		expect(client.delete).toHaveBeenCalledWith('/telegram/link');
	});
});
