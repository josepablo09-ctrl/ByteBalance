import { buildUrl } from './url.js';

/** Código de error cuando no se pudo contactar al servidor (sin respuesta HTTP). */
export const NETWORK_ERROR = 'network_error';

/**
 * Error devuelto por la API con el formato `{ "error": { "code", "message" } }`,
 * o generado por el cliente si la petición no llegó a responderse.
 */
export class ApiError extends Error {
	/**
	 * @param {object} params
	 * @param {number} params.status Código HTTP; 0 si no hubo respuesta.
	 * @param {string} params.code Código estable de la API (p. ej. `invalid_credentials`).
	 * @param {string} params.message Mensaje del servidor.
	 * @param {number | null} [params.retryAfter] Segundos de espera indicados por `Retry-After`.
	 */
	constructor({ status, code, message, retryAfter = null }) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
		this.retryAfter = retryAfter;
	}
}

/**
 * @typedef {object} RequestOptions
 * @property {unknown} [body] Se serializa como JSON.
 * @property {import('./url.js').Query} [query]
 * @property {AbortSignal} [signal]
 */

/**
 * @typedef {object} ClientOptions
 * @property {string} baseUrl URL base normalizada de la API.
 * @property {() => string | null} [getToken] Devuelve el token de acceso vigente, si lo hay.
 * @property {() => void} [onUnauthorized] Se invoca si la API rechaza (401) el token enviado.
 * @property {typeof fetch} [fetch] Implementación de fetch (inyectable en tests).
 */

/**
 * Crea el cliente HTTP de la API REST. Envía `Authorization: Bearer <token>`
 * cuando hay sesión, serializa los cuerpos como JSON y convierte toda
 * respuesta no exitosa en un {@link ApiError}.
 *
 * @param {ClientOptions} options
 */
export function createApiClient({
	baseUrl,
	getToken = () => null,
	onUnauthorized = () => {},
	fetch: fetchFn = (...args) => globalThis.fetch(...args)
}) {
	/**
	 * @param {string} method
	 * @param {string} path
	 * @param {RequestOptions} [options]
	 * @returns {Promise<any>} El cuerpo JSON de la respuesta, o `null` si no tiene.
	 */
	async function request(method, path, { body, query, signal } = {}) {
		/** @type {Record<string, string>} */
		const headers = { Accept: 'application/json' };
		if (body !== undefined) headers['Content-Type'] = 'application/json';
		const token = getToken();
		if (token) headers.Authorization = `Bearer ${token}`;

		let response;
		try {
			response = await fetchFn(buildUrl(baseUrl, path, query), {
				method,
				headers,
				body: body === undefined ? undefined : JSON.stringify(body),
				signal
			});
		} catch (err) {
			// Una cancelación explícita no es un fallo de red: se propaga tal cual.
			if (err instanceof DOMException && err.name === 'AbortError') throw err;
			throw new ApiError({
				status: 0,
				code: NETWORK_ERROR,
				message: 'no se pudo conectar con el servidor'
			});
		}

		const payload = await readJson(response);
		if (response.ok) return payload;

		// Solo un token rechazado invalida la sesión; un 401 de login (credenciales
		// incorrectas) se envía sin token y no debe cerrar nada.
		if (response.status === 401 && token) onUnauthorized();
		throw toApiError(response, payload);
	}

	return {
		request,
		/** @param {string} path @param {Omit<RequestOptions, 'body'>} [options] */
		get: (path, options) => request('GET', path, options),
		/** @param {string} path @param {unknown} [body] @param {Omit<RequestOptions, 'body'>} [options] */
		post: (path, body, options) => request('POST', path, { ...options, body }),
		/** @param {string} path @param {unknown} [body] @param {Omit<RequestOptions, 'body'>} [options] */
		patch: (path, body, options) => request('PATCH', path, { ...options, body }),
		/** @param {string} path @param {Omit<RequestOptions, 'body'>} [options] */
		delete: (path, options) => request('DELETE', path, options)
	};
}

/** @typedef {ReturnType<typeof createApiClient>} ApiClient */

/**
 * Lee el cuerpo como JSON; devuelve `null` si está vacío o no es JSON.
 *
 * @param {Response} response
 */
async function readJson(response) {
	if (response.status === 204) return null;
	const text = await response.text();
	if (!text) return null;
	try {
		return JSON.parse(text);
	} catch {
		return null;
	}
}

/**
 * @param {Response} response
 * @param {any} payload
 */
function toApiError(response, payload) {
	const detail = payload?.error;
	const retryAfter = Number.parseInt(response.headers.get('Retry-After') ?? '', 10);
	return new ApiError({
		status: response.status,
		code: typeof detail?.code === 'string' ? detail.code : `http_${response.status}`,
		message: typeof detail?.message === 'string' ? detail.message : response.statusText,
		retryAfter: Number.isFinite(retryAfter) ? retryAfter : null
	});
}
