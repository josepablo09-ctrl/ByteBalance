import { describe, expect, it, vi } from 'vitest';
import { createAuthApi } from './auth.js';

describe('createAuthApi', () => {
	it('usa los endpoints de autenticación y perfil', async () => {
		const client = {
			request: vi.fn(),
			get: vi.fn(async () => ({ id: 'u1' })),
			post: vi.fn(async () => ({})),
			patch: vi.fn(async () => ({ id: 'u1', monthly_savings_goal_cents: null })),
			delete: vi.fn()
		};
		const api = createAuthApi(client);

		await api.login('ana@example.com', 'secreta');
		await api.me();
		await api.updateMe({ monthly_savings_goal_cents: null });

		expect(client.post).toHaveBeenCalledWith('/auth/login', {
			email: 'ana@example.com',
			password: 'secreta'
		});
		expect(client.get).toHaveBeenCalledWith('/users/me');
		expect(client.patch).toHaveBeenCalledWith('/users/me', { monthly_savings_goal_cents: null });
	});
});
