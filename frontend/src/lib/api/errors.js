import { ApiError, NETWORK_ERROR } from './client.js';

const GENERIC_MESSAGE = 'Ocurrió un error inesperado. Inténtalo de nuevo en unos minutos.';

/**
 * Traduce un error de la API a un mensaje apto para mostrar al usuario. Los
 * errores de validación (400/422) usan el detalle del servidor, que ya es
 * público; los 5xx y los errores desconocidos se muestran de forma genérica.
 *
 * @param {unknown} err
 * @returns {string}
 */
export function errorMessage(err) {
	if (!(err instanceof ApiError)) return GENERIC_MESSAGE;

	switch (err.code) {
		case NETWORK_ERROR:
			return 'No se pudo conectar con el servidor. Revisa tu conexión e inténtalo de nuevo.';
		case 'invalid_credentials':
			return 'Email o contraseña incorrectos.';
		case 'email_taken':
			return 'Ya existe una cuenta con este email.';
		case 'too_many_requests':
			return err.retryAfter
				? `Demasiados intentos. Espera ${formatWait(err.retryAfter)} antes de volver a intentarlo.`
				: 'Demasiados intentos. Espera un momento antes de volver a intentarlo.';
		case 'unauthorized':
			return 'Tu sesión expiró. Inicia sesión de nuevo.';
		case 'ai_disabled':
			return 'El análisis inteligente aún no está disponible en este servidor.';
		case 'ai_unavailable':
			return 'El servicio de IA está saturado o no responde. Inténtalo de nuevo en unos minutos.';
	}
	if ((err.status === 400 || err.status === 422) && err.message) {
		return capitalize(err.message) + (err.message.endsWith('.') ? '' : '.');
	}
	return GENERIC_MESSAGE;
}

/** @param {number} seconds */
function formatWait(seconds) {
	if (seconds < 60) return `${seconds} s`;
	const minutes = Math.ceil(seconds / 60);
	return minutes === 1 ? '1 minuto' : `${minutes} minutos`;
}

/** @param {string} s */
function capitalize(s) {
	return s.charAt(0).toUpperCase() + s.slice(1);
}
