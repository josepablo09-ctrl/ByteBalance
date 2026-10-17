/**
 * @typedef {object} AIStatus
 * @property {boolean} enabled `false` si el servidor no tiene IA configurada.
 */

/**
 * @typedef {object} Suggestion
 * @property {string} title
 * @property {string} detail
 */

/**
 * Diagnóstico generado por la IA. Todo el texto es plano (sin HTML ni Markdown).
 *
 * @typedef {object} Diagnosis
 * @property {string} headline
 * @property {string} summary Uno o más párrafos separados por líneas en blanco.
 * @property {string[]} finance_insights
 * @property {string[]} habit_insights
 * @property {Suggestion[]} suggestions
 * @property {{ from: string, to: string }} period Fechas YYYY-MM-DD (inclusive).
 * @property {string} generated_at Fecha RFC 3339.
 */

/**
 * Endpoints del análisis inteligente.
 *
 * @param {import('./client.js').ApiClient} client
 */
export function createAIApi(client) {
	return {
		/** @returns {Promise<AIStatus>} */
		status: () => client.get('/ai'),

		/** @returns {Promise<Diagnosis>} */
		diagnose: () => client.post('/ai/diagnose')
	};
}

/** @typedef {ReturnType<typeof createAIApi>} AIApi */
