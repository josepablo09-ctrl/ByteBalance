/**
 * Valida y normaliza la URL base de la API: debe ser http(s) y se devuelve sin
 * barra final para poder concatenar rutas como `/auth/login`.
 *
 * @param {string | undefined} raw
 * @returns {string}
 */
export function normalizeBaseUrl(raw) {
	const value = (raw ?? '').trim();
	if (!value) {
		throw new Error('VITE_API_URL no está definida: copia frontend/.env.example a frontend/.env');
	}
	let url;
	try {
		url = new URL(value);
	} catch {
		throw new Error(`VITE_API_URL no es una URL válida: ${value}`);
	}
	if (url.protocol !== 'http:' && url.protocol !== 'https:') {
		throw new Error(`VITE_API_URL debe usar http o https: ${value}`);
	}
	return value.replace(/\/+$/, '');
}

/** @typedef {Record<string, string | number | boolean | null | undefined>} Query */

/**
 * Construye la URL de un endpoint. Los parámetros nulos o indefinidos se omiten.
 *
 * @param {string} baseUrl URL base ya normalizada (sin barra final).
 * @param {string} path Ruta del endpoint, p. ej. `/habits`.
 * @param {Query} [query]
 * @returns {string}
 */
export function buildUrl(baseUrl, path, query) {
	const url = baseUrl + (path.startsWith('/') ? path : `/${path}`);
	if (!query) return url;
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(query)) {
		if (value !== null && value !== undefined) params.append(key, String(value));
	}
	const qs = params.toString();
	return qs ? `${url}?${qs}` : url;
}
