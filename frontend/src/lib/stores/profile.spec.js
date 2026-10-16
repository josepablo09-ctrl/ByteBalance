import { get } from 'svelte/store';
import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '#lib/api/client.js';
import { createProfileStore } from './profile.js';

const user = /** @type {import('#lib/api/auth.js').User} */ ({ id: 'u1', full_name: 'Ana Pérez' });

/** Promesa controlable desde el test. */
function deferred() {
	/** @type {(value: any) => void} */
	let resolve = () => {};
	/** @type {(reason: unknown) => void} */
	let reject = () => {};
	const promise = new Promise((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

describe('createProfileStore', () => {
	it('carga el perfil', async () => {
		const store = createProfileStore(async () => user);
		expect(get(store).status).toBe('idle');

		const loading = store.load();
		expect(get(store).status).toBe('loading');
		await loading;
		expect(get(store)).toEqual({ status: 'ready', user, error: '' });
	});

	it('comparte una sola petición entre llamadas simultáneas', async () => {
		const fetchProfile = vi.fn(async () => user);
		const store = createProfileStore(fetchProfile);
		await Promise.all([store.load(), store.load()]);
		expect(fetchProfile).toHaveBeenCalledOnce();
	});

	it('expone un mensaje si falla', async () => {
		const store = createProfileStore(async () => {
			throw new ApiError({ status: 0, code: 'network_error', message: 'x' });
		});
		await store.load();
		expect(get(store).status).toBe('error');
		expect(get(store).error).toMatch(/No se pudo conectar/);
	});

	it('descarta una respuesta que llega después de cerrar sesión', async () => {
		const pending = deferred();
		const store = createProfileStore(() => pending.promise);
		const loading = store.load();

		store.reset();
		pending.resolve(user);
		await loading;

		expect(get(store)).toEqual({ status: 'idle', user: null, error: '' });
	});
});
