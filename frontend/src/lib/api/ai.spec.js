import { describe, expect, it, vi } from 'vitest';
import { createAIApi } from './ai.js';

describe('createAIApi', () => {
	it('usa los endpoints de IA', async () => {
		const client = {
			request: vi.fn(),
			get: vi.fn(async () => ({ enabled: true })),
			post: vi.fn(async () => ({ headline: 'Vas bien' })),
			patch: vi.fn(),
			delete: vi.fn()
		};
		const api = createAIApi(client);

		await expect(api.status()).resolves.toEqual({ enabled: true });
		await expect(api.diagnose()).resolves.toEqual({ headline: 'Vas bien' });

		expect(client.get).toHaveBeenCalledWith('/ai');
		expect(client.post).toHaveBeenCalledWith('/ai/diagnose');
	});
});
