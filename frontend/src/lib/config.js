import { normalizeBaseUrl } from '#lib/api/url.js';

/**
 * URL base de la API REST (VITE_API_URL). Las variables VITE_* son públicas:
 * se incrustan en el bundle del navegador y nunca deben contener secretos. En
 * un build de producción debe usar https.
 */
export const API_URL = normalizeBaseUrl(import.meta.env.VITE_API_URL, {
	requireHttps: import.meta.env.PROD
});
