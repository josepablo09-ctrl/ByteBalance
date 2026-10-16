import { LOCALE } from './locale.js';

/** Monto máximo admitido por el backend (`finance.MaxCents`, NUMERIC(14,2)). */
export const MAX_CENTS = 99_999_999_999_999;

/**
 * Convierte centavos en un decimal exacto en texto (p. ej. -123456 → "-1234.56").
 * Evita dividir entre 100 en coma flotante.
 *
 * @param {number} cents Entero.
 */
export function centsToDecimal(cents) {
	const abs = Math.abs(cents);
	const units = (abs - (abs % 100)) / 100;
	const fraction = String(abs % 100).padStart(2, '0');
	return `${cents < 0 ? '-' : ''}${units}.${fraction}`;
}

/** @type {Map<string, Intl.NumberFormat>} */
const formatters = new Map();

/**
 * Formatea centavos como moneda, p. ej. `USD 1,234.56`. Se muestra el código
 * ISO y no el símbolo, porque `$` sería ambiguo entre USD, MXN, COP, etc.
 *
 * @param {number} cents
 * @param {string} currency Código ISO 4217.
 * @param {{ signDisplay?: 'auto' | 'always' | 'exceptZero' }} [options]
 */
export function formatMoney(cents, currency, { signDisplay = 'auto' } = {}) {
	const key = `${currency}|${signDisplay}`;
	let formatter = formatters.get(key);
	if (!formatter) {
		formatter = new Intl.NumberFormat(LOCALE, {
			style: 'currency',
			currency,
			currencyDisplay: 'code',
			signDisplay,
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		});
		formatters.set(key, formatter);
	}
	// format() acepta un decimal en texto: así no hay pérdida de precisión.
	return formatter.format(/** @type {Intl.StringNumericLiteral} */ (centsToDecimal(cents)));
}

/**
 * Interpreta el monto que escribe el usuario: dígitos con un separador decimal
 * opcional (punto o coma) y hasta dos decimales. No admite separadores de
 * miles, para que "1.250" no se lea como mil doscientos cincuenta.
 *
 * @param {string} input
 * @returns {{ cents: number, error?: undefined } | { cents?: undefined, error: string }}
 */
export function parseAmount(input) {
	const value = input.trim().replace(/\s+/g, '');
	if (!value) return { error: 'Ingresa un monto.' };
	const match = /^(\d{1,12})(?:[.,](\d{1,2}))?$/.exec(value);
	if (!match) return { error: 'Ingresa un monto válido, p. ej. 1250.50' };

	const cents = Number(match[1]) * 100 + Number((match[2] ?? '').padEnd(2, '0'));
	if (cents <= 0) return { error: 'El monto debe ser mayor que cero.' };
	if (cents > MAX_CENTS) return { error: 'El monto es demasiado grande.' };
	return { cents };
}
