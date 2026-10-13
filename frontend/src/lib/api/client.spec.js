import { describe, expect, it, vi } from 'vitest';
import { ApiError, createApiClient, NETWORK_ERROR } from './client.js';

const BASE_URL = 'http://localhost:8080/api/v1';

/**
 * @param {number} status
 * @param {unknown} [body]
 * @param {Record<string, string>} [headers]
 */
function jsonResponse(status, body, headers = {}) {
	return new Response(body === undefined ? null : JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json', ...headers }
	});
}

/**
 * Crea un cliente con un fetch simulado que responde `response`.
 *
 * @param {Response | (() => Promise<Response>)} response
 * @param {Partial<import('./client.js').ClientOptions>} [options]
 */
function setup(response, options = {}) {
	/** @type {(input: RequestInfo | URL, init?: RequestInit) => Promise<Response>} */
	const respond = typeof response === 'function' ? response : async () => response;
	const fetchMock = vi.fn(respond);
	const client = createApiClient({ baseUrl: BASE_URL, fetch: fetchMock, ...options });
	// El cliente siempre llama a fetch con una URL en texto y un RequestInit.
	const lastCall = () => {
		const call = fetchMock.mock.lastCall;
		if (!call) throw new Error('fetch no fue llamado');
		return /** @type {[string, RequestInit]} */ (call);
	};
	return { client, fetchMock, lastCall };
}

describe('createApiClient', () => {
	it('envía el Bearer token cuando hay sesión', async () => {
		const { client, lastCall } = setup(jsonResponse(200, { id: '1' }), {
			getToken: () => 'jwt-123'
		});

		await expect(client.get('/users/me')).resolves.toEqual({ id: '1' });

		const [url, init] = lastCall();
		expect(url).toBe(`${BASE_URL}/users/me`);
		expect(init.method).toBe('GET');
		expect(init.headers).toMatchObject({
			Authorization: 'Bearer jwt-123',
			Accept: 'application/json'
		});
		expect(init.body).toBeUndefined();
	});

	it('no envía Authorization sin sesión', async () => {
		const { client, lastCall } = setup(jsonResponse(200, {}));
		await client.get('/habits');
		expect(lastCall()[1].headers).not.toHaveProperty('Authorization');
	});

	it('serializa el cuerpo como JSON', async () => {
		const { client, lastCall } = setup(jsonResponse(201, { id: '1' }));
		await client.post('/users', { email: 'ana@example.com' });

		const [, init] = lastCall();
		expect(init.method).toBe('POST');
		expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' });
		expect(init.body).toBe('{"email":"ana@example.com"}');
	});

	it('añade los parámetros de consulta', async () => {
		const { client, lastCall } = setup(jsonResponse(200, { habits: [] }));
		await client.get('/habits', { query: { active: true } });
		expect(lastCall()[0]).toBe(`${BASE_URL}/habits?active=true`);
	});

	it('devuelve null en respuestas sin cuerpo', async () => {
		const { client } = setup(new Response(null, { status: 204 }));
		await expect(client.delete('/habits/1')).resolves.toBeNull();
	});

	it('convierte el error de la API en ApiError', async () => {
		const { client } = setup(
			jsonResponse(409, { error: { code: 'email_taken', message: 'el email ya está registrado' } })
		);

		const err = await client.post('/users', {}).catch((e) => e);
		expect(err).toBeInstanceOf(ApiError);
		expect(err).toMatchObject({
			status: 409,
			code: 'email_taken',
			message: 'el email ya está registrado'
		});
	});

	it('lee Retry-After en un 429', async () => {
		const { client } = setup(
			jsonResponse(
				429,
				{ error: { code: 'too_many_requests', message: 'demasiados intentos' } },
				{ 'Retry-After': '42' }
			)
		);
		const err = await client.post('/auth/login', {}).catch((e) => e);
		expect(err).toMatchObject({ status: 429, code: 'too_many_requests', retryAfter: 42 });
	});

	it('tolera errores sin cuerpo JSON', async () => {
		const { client } = setup(new Response('<html>Bad Gateway</html>', { status: 502 }));
		const err = await client.get('/users/me').catch((e) => e);
		expect(err).toMatchObject({ status: 502, code: 'http_502', retryAfter: null });
	});

	it('cierra la sesión si la API rechaza el token', async () => {
		const onUnauthorized = vi.fn();
		const { client } = setup(jsonResponse(401, { error: { code: 'unauthorized', message: 'x' } }), {
			getToken: () => 'jwt-expirado',
			onUnauthorized
		});

		await expect(client.get('/users/me')).rejects.toMatchObject({ status: 401 });
		expect(onUnauthorized).toHaveBeenCalledOnce();
	});

	it('no cierra la sesión por un 401 sin token (credenciales incorrectas)', async () => {
		const onUnauthorized = vi.fn();
		const { client } = setup(
			jsonResponse(401, { error: { code: 'invalid_credentials', message: 'x' } }),
			{ onUnauthorized }
		);

		await expect(client.post('/auth/login', {})).rejects.toMatchObject({
			code: 'invalid_credentials'
		});
		expect(onUnauthorized).not.toHaveBeenCalled();
	});

	it('reporta un fallo de red como ApiError con status 0', async () => {
		const { client } = setup(async () => {
			throw new TypeError('Failed to fetch');
		});
		await expect(client.get('/users/me')).rejects.toMatchObject({ status: 0, code: NETWORK_ERROR });
	});

	it('propaga las cancelaciones sin convertirlas', async () => {
		const { client } = setup(async () => {
			throw new DOMException('aborted', 'AbortError');
		});
		await expect(client.get('/users/me')).rejects.toMatchObject({ name: 'AbortError' });
	});
});
