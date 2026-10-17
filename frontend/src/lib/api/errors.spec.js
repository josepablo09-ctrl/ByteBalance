import { describe, expect, it } from 'vitest';
import { ApiError, NETWORK_ERROR } from './client.js';
import { errorMessage } from './errors.js';

/** @param {Partial<ConstructorParameters<typeof ApiError>[0]>} params */
const apiError = (params) => new ApiError({ status: 400, code: 'x', message: '', ...params });

describe('errorMessage', () => {
	it.each([
		[apiError({ status: 0, code: NETWORK_ERROR }), /No se pudo conectar/],
		[apiError({ status: 401, code: 'invalid_credentials' }), 'Email o contraseña incorrectos.'],
		[apiError({ status: 409, code: 'email_taken' }), 'Ya existe una cuenta con este email.'],
		[apiError({ status: 429, code: 'too_many_requests', retryAfter: 42 }), /Espera 42 s/],
		[apiError({ status: 429, code: 'too_many_requests', retryAfter: 60 }), /Espera 1 minuto /],
		[apiError({ status: 429, code: 'too_many_requests', retryAfter: 150 }), /Espera 3 minutos/],
		[apiError({ status: 429, code: 'too_many_requests' }), /Espera un momento/],
		[apiError({ status: 503, code: 'ai_disabled' }), /aún no está disponible/],
		[apiError({ status: 503, code: 'ai_unavailable' }), /servicio de IA está saturado/],
		[
			apiError({ status: 422, code: 'not_enough_data', message: 'aún no hay datos para analizar' }),
			'Aún no hay datos para analizar.'
		],
		[
			apiError({ status: 422, code: 'invalid_input', message: 'el nombre es obligatorio' }),
			'El nombre es obligatorio.'
		],
		[
			apiError({ status: 500, code: 'internal', message: 'error interno del servidor' }),
			/inesperado/
		],
		[new Error('boom'), /inesperado/],
		['texto', /inesperado/]
	])('traduce %o', (err, want) => {
		const message = errorMessage(err);
		if (typeof want === 'string') expect(message).toBe(want);
		else expect(message).toMatch(want);
	});
});
