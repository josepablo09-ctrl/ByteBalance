import { browser } from '$app/env';
import { get, writable } from 'svelte/store';

export const SESSION_STORAGE_KEY = 'bytebalance.session';

/**
 * @typedef {object} Session
 * @property {string} token Token de acceso (JWT).
 * @property {string} expiresAt Fecha de expiración RFC 3339.
 */

/**
 * Crea el store de sesión. La sesión se persiste en `storage` para sobrevivir a
 * recargas; una sesión expirada o corrupta se descarta al leerla.
 *
 * @param {Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>} [storage]
 * @param {() => number} [now] Reloj en milisegundos (inyectable en tests).
 */
export function createSessionStore(storage, now = Date.now) {
	/** @param {unknown} s @returns {s is Session} */
	const isValid = (s) =>
		typeof s === 'object' &&
		s !== null &&
		'token' in s &&
		typeof s.token === 'string' &&
		s.token !== '' &&
		'expiresAt' in s &&
		typeof s.expiresAt === 'string' &&
		Date.parse(s.expiresAt) > now();

	/** @returns {Session | null} */
	function load() {
		try {
			const raw = storage?.getItem(SESSION_STORAGE_KEY);
			if (!raw) return null;
			const parsed = JSON.parse(raw);
			if (isValid(parsed)) return parsed;
			storage?.removeItem(SESSION_STORAGE_KEY);
		} catch {
			// Almacenamiento bloqueado o contenido corrupto: se empieza sin sesión.
		}
		return null;
	}

	/** @type {import('svelte/store').Writable<Session | null>} */
	const store = writable(load());

	function clear() {
		store.set(null);
		try {
			storage?.removeItem(SESSION_STORAGE_KEY);
		} catch {
			// Sin almacenamiento disponible la sesión solo vivía en memoria.
		}
	}

	/** Devuelve el token vigente, o `null` si no hay sesión o ya expiró. */
	function token() {
		const current = get(store);
		if (!current) return null;
		if (!isValid(current)) {
			clear();
			return null;
		}
		return current.token;
	}

	return {
		subscribe: store.subscribe,
		/** @param {Session} session */
		start(session) {
			store.set(session);
			try {
				storage?.setItem(SESSION_STORAGE_KEY, JSON.stringify(session));
			} catch {
				// Sin almacenamiento la sesión dura hasta que se cierre la pestaña.
			}
		},
		clear,
		token,
		isAuthenticated: () => token() !== null
	};
}

/** Sesión del usuario en este navegador. */
export const session = createSessionStore(browser ? globalThis.localStorage : undefined);
