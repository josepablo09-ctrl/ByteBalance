import { writable } from 'svelte/store';
import { authApi, errorMessage } from '#lib/api/index.js';

/**
 * @typedef {object} ProfileState
 * @property {'idle' | 'loading' | 'ready' | 'error'} status
 * @property {import('#lib/api/auth.js').User | null} user
 * @property {string} error Mensaje para el usuario si `status` es `error`.
 */

/** @type {ProfileState} */
const INITIAL = { status: 'idle', user: null, error: '' };

/**
 * Crea el store del perfil del usuario autenticado, compartido por el header y
 * las pantallas que necesitan su moneda o zona horaria.
 *
 * @param {() => Promise<import('#lib/api/auth.js').User>} fetchProfile
 */
export function createProfileStore(fetchProfile) {
	const store = writable(INITIAL);
	/** @type {Promise<void> | null} */
	let pending = null;
	// Invalida respuestas que lleguen después de un reset (p. ej. tras logout).
	let generation = 0;

	/** Carga el perfil; las llamadas simultáneas comparten la misma petición. */
	function load() {
		if (pending) return pending;
		const current = ++generation;
		store.update((s) => ({ ...s, status: 'loading', error: '' }));
		pending = fetchProfile()
			.then((user) => {
				if (current === generation) store.set({ status: 'ready', user, error: '' });
			})
			.catch((err) => {
				if (current === generation) {
					store.set({ status: 'error', user: null, error: errorMessage(err) });
				}
			})
			.finally(() => {
				if (current === generation) pending = null;
			});
		return pending;
	}

	/** Olvida el perfil (al cerrar sesión), para no mostrarlo a otro usuario. */
	function reset() {
		generation++;
		pending = null;
		store.set(INITIAL);
	}

	/**
	 * Reemplaza el perfil por la versión que devolvió la API tras modificarlo
	 * (p. ej. la meta de ahorro), sin volver a pedirlo.
	 *
	 * @param {import('#lib/api/auth.js').User} user
	 */
	function replace(user) {
		generation++;
		pending = null;
		store.set({ status: 'ready', user, error: '' });
	}

	return { subscribe: store.subscribe, load, reset, replace };
}

/** Perfil del usuario con sesión iniciada. */
export const profile = createProfileStore(() => authApi.me());
