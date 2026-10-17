/**
 * @typedef {object} AccessToken
 * @property {string} access_token
 * @property {'Bearer'} token_type
 * @property {string} expires_at Fecha RFC 3339.
 */

/**
 * @typedef {object} User
 * @property {string} id
 * @property {string} email
 * @property {string} full_name
 * @property {string} default_currency Código ISO 4217.
 * @property {string} timezone Zona horaria IANA.
 * @property {boolean} telegram_linked
 * @property {number | null} monthly_savings_goal_cents Meta de ahorro mensual en
 *   `default_currency`; `null` si no hay meta.
 * @property {string} created_at Fecha RFC 3339.
 */

/**
 * @typedef {object} RegisterInput
 * @property {string} email
 * @property {string} password
 * @property {string} full_name
 * @property {string} [default_currency]
 * @property {string} [timezone]
 */

/**
 * Endpoints de autenticación y perfil.
 *
 * @param {import('./client.js').ApiClient} client
 */
export function createAuthApi(client) {
	return {
		/**
		 * @param {string} email
		 * @param {string} password
		 * @returns {Promise<AccessToken>}
		 */
		login: (email, password) => client.post('/auth/login', { email, password }),

		/**
		 * @param {RegisterInput} input
		 * @returns {Promise<User>}
		 */
		register: (input) => client.post('/users', input),

		/** @returns {Promise<User>} */
		me: () => client.get('/users/me'),

		/**
		 * Cambia preferencias del perfil; solo cambian los campos enviados y
		 * `monthly_savings_goal_cents: null` quita la meta de ahorro.
		 *
		 * @param {{ monthly_savings_goal_cents?: number | null }} changes
		 * @returns {Promise<User>}
		 */
		updateMe: (changes) => client.patch('/users/me', changes)
	};
}

/** @typedef {ReturnType<typeof createAuthApi>} AuthApi */
