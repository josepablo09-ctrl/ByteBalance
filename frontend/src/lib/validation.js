// Reglas espejo de las del backend (internal/domain/user y internal/usecase/user.go).
// Validar en el cliente solo mejora la experiencia: el servidor vuelve a validar todo.
export const MIN_PASSWORD_LENGTH = 8;
export const MAX_PASSWORD_BYTES = 72;
export const MAX_EMAIL_LENGTH = 255;
export const MAX_FULL_NAME_LENGTH = 150;

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const CURRENCY_PATTERN = /^[A-Z]{3}$/;
const encoder = new TextEncoder();

/** @typedef {Partial<Record<string, string>>} FieldErrors */

/** @param {string} email @returns {string | undefined} */
export function validateEmail(email) {
	const value = email.trim();
	if (!value) return 'Ingresa tu email.';
	if (value.length > MAX_EMAIL_LENGTH || !EMAIL_PATTERN.test(value))
		return 'Ingresa un email válido.';
}

/**
 * La longitud mínima se cuenta en caracteres y la máxima en bytes UTF-8, igual
 * que en el backend (bcrypt no admite más de 72 bytes).
 *
 * @param {string} password
 * @returns {string | undefined}
 */
export function validateNewPassword(password) {
	if (!password) return 'Ingresa una contraseña.';
	if ([...password].length < MIN_PASSWORD_LENGTH) {
		return `Usa al menos ${MIN_PASSWORD_LENGTH} caracteres.`;
	}
	if (encoder.encode(password).length > MAX_PASSWORD_BYTES) {
		return 'La contraseña es demasiado larga.';
	}
}

/**
 * @param {{ email: string, password: string }} input
 * @returns {FieldErrors}
 */
export function validateLogin({ email, password }) {
	return compact({
		email: validateEmail(email),
		password: password ? undefined : 'Ingresa tu contraseña.'
	});
}

/**
 * @param {{ fullName: string, email: string, password: string, currency: string }} input
 * @returns {FieldErrors}
 */
export function validateRegister({ fullName, email, password, currency }) {
	const name = fullName.trim();
	return compact({
		fullName: !name
			? 'Ingresa tu nombre.'
			: [...name].length > MAX_FULL_NAME_LENGTH
				? `Usa como máximo ${MAX_FULL_NAME_LENGTH} caracteres.`
				: undefined,
		email: validateEmail(email),
		password: validateNewPassword(password),
		currency: CURRENCY_PATTERN.test(currency) ? undefined : 'Elige una moneda.'
	});
}

/** @param {Record<string, string | undefined>} errors @returns {FieldErrors} */
function compact(errors) {
	return Object.fromEntries(Object.entries(errors).filter(([, message]) => message));
}

/** @param {FieldErrors} errors */
export function hasErrors(errors) {
	return Object.keys(errors).length > 0;
}

/**
 * Zona horaria IANA del navegador (p. ej. `America/Costa_Rica`); `UTC` si no
 * se puede determinar.
 */
export function browserTimezone() {
	try {
		return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
	} catch {
		return 'UTC';
	}
}
