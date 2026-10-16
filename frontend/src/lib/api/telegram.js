/**
 * @typedef {object} TelegramStatus
 * @property {boolean} enabled `false` si el servidor no tiene bot configurado.
 * @property {boolean} linked
 * @property {string} [bot_username] Sin "@".
 */

/**
 * @typedef {object} TelegramLink
 * @property {string} code Código de un solo uso; también vale enviar `/start <code>` al bot.
 * @property {string} url Enlace `https://t.me/<bot>?start=<code>`.
 * @property {string} expires_at Fecha RFC 3339.
 */

/**
 * Endpoints de la vinculación con Telegram.
 *
 * @param {import('./client.js').ApiClient} client
 */
export function createTelegramApi(client) {
	return {
		/** @returns {Promise<TelegramStatus>} */
		status: () => client.get('/telegram'),

		/** @returns {Promise<TelegramLink>} */
		createLink: () => client.post('/telegram/link'),

		/** @returns {Promise<void>} */
		async unlink() {
			await client.delete('/telegram/link');
		},

		/** @returns {Promise<void>} */
		async sendTest() {
			await client.post('/telegram/test');
		}
	};
}

/** @typedef {ReturnType<typeof createTelegramApi>} TelegramApi */
