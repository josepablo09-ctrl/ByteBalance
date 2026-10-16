/** Monedas ofrecidas en los formularios. El backend admite cualquier código ISO 4217. */
export const CURRENCIES = [
	{ value: 'USD', label: 'USD · Dólar estadounidense' },
	{ value: 'EUR', label: 'EUR · Euro' },
	{ value: 'CRC', label: 'CRC · Colón costarricense' },
	{ value: 'MXN', label: 'MXN · Peso mexicano' },
	{ value: 'COP', label: 'COP · Peso colombiano' },
	{ value: 'ARS', label: 'ARS · Peso argentino' },
	{ value: 'CLP', label: 'CLP · Peso chileno' },
	{ value: 'PEN', label: 'PEN · Sol peruano' },
	{ value: 'GBP', label: 'GBP · Libra esterlina' }
];

/**
 * Opciones de moneda que siempre incluyen `current`, aunque no esté en la lista
 * (p. ej. un usuario registrado con otra moneda).
 *
 * @param {string} current
 */
export function currencyOptions(current) {
	if (!current || CURRENCIES.some((c) => c.value === current)) return CURRENCIES;
	return [{ value: current, label: current }, ...CURRENCIES];
}
