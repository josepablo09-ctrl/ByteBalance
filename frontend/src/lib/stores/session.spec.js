import { get } from 'svelte/store';
import { describe, expect, it } from 'vitest';
import { createSessionStore, SESSION_STORAGE_KEY } from './session.js';

/** Storage en memoria compatible con localStorage. */
function memoryStorage(initial = /** @type {Record<string, string>} */ ({})) {
	const data = new Map(Object.entries(initial));
	return {
		data,
		/** @param {string} key */
		getItem: (key) => data.get(key) ?? null,
		/** @param {string} key @param {string} value */
		setItem: (key, value) => void data.set(key, value),
		/** @param {string} key */
		removeItem: (key) => void data.delete(key)
	};
}

const NOW = Date.parse('2026-10-13T12:00:00Z');
const valid = { token: 'jwt-123', expiresAt: '2026-10-14T12:00:00Z' };
const expired = { token: 'jwt-viejo', expiresAt: '2026-10-13T11:59:59Z' };

describe('createSessionStore', () => {
	it('empieza sin sesión', () => {
		const s = createSessionStore(memoryStorage(), () => NOW);
		expect(get(s)).toBeNull();
		expect(s.token()).toBeNull();
		expect(s.isAuthenticated()).toBe(false);
	});

	it('persiste la sesión iniciada y la restaura al recargar', () => {
		const storage = memoryStorage();
		createSessionStore(storage, () => NOW).start(valid);

		const reloaded = createSessionStore(storage, () => NOW);
		expect(get(reloaded)).toEqual(valid);
		expect(reloaded.token()).toBe('jwt-123');
	});

	it('clear borra la sesión en memoria y en el almacenamiento', () => {
		const storage = memoryStorage();
		const s = createSessionStore(storage, () => NOW);
		s.start(valid);
		s.clear();
		expect(get(s)).toBeNull();
		expect(storage.data.has(SESSION_STORAGE_KEY)).toBe(false);
	});

	it.each([
		['expirada', JSON.stringify(expired)],
		['JSON corrupto', '{no-json'],
		['sin token', JSON.stringify({ expiresAt: valid.expiresAt })],
		['fecha inválida', JSON.stringify({ token: 'x', expiresAt: 'mañana' })]
	])('descarta una sesión guardada %s', (_, raw) => {
		const storage = memoryStorage({ [SESSION_STORAGE_KEY]: raw });
		const s = createSessionStore(storage, () => NOW);
		expect(get(s)).toBeNull();
	});

	it('invalida la sesión cuando expira mientras la app está abierta', () => {
		let now = NOW;
		const storage = memoryStorage();
		const s = createSessionStore(storage, () => now);
		s.start(valid);
		expect(s.isAuthenticated()).toBe(true);

		now = Date.parse(valid.expiresAt);
		expect(s.token()).toBeNull();
		expect(get(s)).toBeNull();
		expect(storage.data.has(SESSION_STORAGE_KEY)).toBe(false);
	});

	it('funciona sin almacenamiento disponible', () => {
		const throwing = {
			getItem: () => {
				throw new Error('bloqueado');
			},
			setItem: () => {
				throw new Error('bloqueado');
			},
			removeItem: () => {
				throw new Error('bloqueado');
			}
		};
		const s = createSessionStore(throwing, () => NOW);
		s.start(valid);
		expect(s.token()).toBe('jwt-123');
		s.clear();
		expect(s.token()).toBeNull();
	});
});
