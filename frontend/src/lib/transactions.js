import { zonedToInstant } from './dates.js';
import { parseAmount } from './money.js';

// Igual que maxDescriptionLen y maxCategoryNameLen en internal/domain/finance.
export const MAX_DESCRIPTION_LENGTH = 255;
export const MAX_CATEGORY_NAME_LENGTH = 80;

/** Hora del día asignada a una transacción fechada en un día pasado o futuro. */
const DEFAULT_HOUR = 12;

/**
 * @typedef {object} TransactionDraft Valores del formulario de transacción.
 * @property {import('./api/finance.js').EntryType} type
 * @property {string} amount Monto tal como lo escribió el usuario.
 * @property {string} currency
 * @property {string} categoryId Vacío = sin categoría.
 * @property {string} description
 * @property {string} date `YYYY-MM-DD` en la zona horaria del usuario.
 */

/**
 * Valida el formulario y construye la petición para la API.
 *
 * Al crear, una transacción de hoy se envía sin `occurred_at` para que el
 * backend use la hora actual; una de otro día se fecha a las 12:00 en la zona
 * del usuario, lo que la mantiene en ese día aunque se consulte desde otra zona.
 *
 * Al editar (`originalDate`), la fecha solo se envía si cambió, para conservar
 * la hora original; si se mueve a hoy se usa la hora actual (`now`) en lugar de
 * las 12:00, que podría quedar en el futuro.
 *
 * @param {TransactionDraft} draft
 * @param {{ timeZone: string, today: string, originalDate?: string, now?: Date }} context
 * @returns {{ input: import('./api/finance.js').RecordTransactionInput, errors?: undefined }
 *   | { input?: undefined, errors: Partial<Record<keyof TransactionDraft, string>> }}
 */
export function buildTransactionInput(draft, { timeZone, today, originalDate, now = new Date() }) {
	/** @type {Partial<Record<keyof TransactionDraft, string>>} */
	const errors = {};

	const amount = parseAmount(draft.amount);
	if (amount.error) errors.amount = amount.error;
	if (!/^[A-Z]{3}$/.test(draft.currency)) errors.currency = 'Elige una moneda.';
	const description = draft.description.trim();
	if ([...description].length > MAX_DESCRIPTION_LENGTH) {
		errors.description = `Usa como máximo ${MAX_DESCRIPTION_LENGTH} caracteres.`;
	}
	const date = /^(\d{4})-(\d{2})-(\d{2})$/.exec(draft.date);
	if (!date) errors.date = 'Elige una fecha.';

	if (Object.keys(errors).length > 0 || amount.cents === undefined || !date) return { errors };

	/** @type {import('./api/finance.js').RecordTransactionInput} */
	const input = {
		type: draft.type,
		amount_cents: amount.cents,
		currency: draft.currency,
		category_id: draft.categoryId || null,
		description
	};
	if (originalDate !== undefined) {
		if (draft.date === originalDate) return { input };
		if (draft.date === today) {
			input.occurred_at = now.toISOString();
			return { input };
		}
	}
	if (draft.date !== today) {
		const [, year, month, day] = date.map(Number);
		input.occurred_at = zonedToInstant(
			{ year, month, day, hour: DEFAULT_HOUR },
			timeZone
		).toISOString();
	}
	return { input };
}

/**
 * @param {string} name
 * @returns {string | undefined} Mensaje de error, si el nombre no es válido.
 */
export function validateCategoryName(name) {
	const value = name.trim();
	if (!value) return 'Escribe un nombre.';
	if ([...value].length > MAX_CATEGORY_NAME_LENGTH) {
		return `Usa como máximo ${MAX_CATEGORY_NAME_LENGTH} caracteres.`;
	}
}
