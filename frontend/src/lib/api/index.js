import { API_URL } from '#lib/config.js';
import { session } from '#lib/stores/session.js';
import { createAIApi } from './ai.js';
import { createAuthApi } from './auth.js';
import { createApiClient } from './client.js';
import { createFinanceApi } from './finance.js';
import { createHabitsApi } from './habits.js';
import { createTelegramApi } from './telegram.js';

export { ApiError, NETWORK_ERROR } from './client.js';
export { errorMessage } from './errors.js';

/** Cliente HTTP de la app: adjunta el token de la sesión y la cierra si la API lo rechaza. */
export const api = createApiClient({
	baseUrl: API_URL,
	getToken: () => session.token(),
	onUnauthorized: () => session.clear()
});

export const authApi = createAuthApi(api);
export const financeApi = createFinanceApi(api);
export const habitsApi = createHabitsApi(api);
export const telegramApi = createTelegramApi(api);
export const aiApi = createAIApi(api);
