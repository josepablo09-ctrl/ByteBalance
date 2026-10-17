import { writable } from 'svelte/store';

/**
 * Último diagnóstico de IA de la sesión. Se conserva al cambiar de página para
 * no gastar otro análisis (que tiene límite) solo por volver al panel.
 */
function createInsightStore() {
	const store = writable(/** @type {import('#lib/api/ai.js').Diagnosis | null} */ (null));
	return {
		subscribe: store.subscribe,
		/** @param {import('#lib/api/ai.js').Diagnosis} diagnosis */
		set: (diagnosis) => store.set(diagnosis),
		/** Olvida el diagnóstico (al cerrar sesión), para no mostrarlo a otro usuario. */
		reset: () => store.set(null)
	};
}

export const lastInsight = createInsightStore();
