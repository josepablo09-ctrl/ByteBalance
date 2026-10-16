import { describe, expect, it } from 'vitest';
import { centsToDecimal, formatMoney, MAX_CENTS, parseAmount } from './money.js';

describe('centsToDecimal', () => {
	it.each([
		[0, '0.00'],
		[5, '0.05'],
		[123456, '1234.56'],
		[-123456, '-1234.56'],
		[MAX_CENTS, '999999999999.99']
	])('%d → %s', (cents, want) => {
		expect(centsToDecimal(cents)).toBe(want);
	});
});

describe('formatMoney', () => {
	// Intl separa el código del número con un espacio duro (U+00A0).
	const NBSP = ' ';

	it('usa el código ISO y dos decimales', () => {
		expect(formatMoney(123456, 'USD')).toBe(`USD${NBSP}1,234.56`);
		expect(formatMoney(50, 'CRC')).toBe(`CRC${NBSP}0.50`);
	});

	it('conserva la precisión del monto máximo', () => {
		expect(formatMoney(MAX_CENTS, 'USD')).toBe(`USD${NBSP}999,999,999,999.99`);
	});

	it('muestra el signo cuando se pide', () => {
		expect(formatMoney(1250, 'EUR', { signDisplay: 'always' })).toBe(`+EUR${NBSP}12.50`);
		expect(formatMoney(-1250, 'EUR', { signDisplay: 'always' })).toBe(`-EUR${NBSP}12.50`);
	});
});

describe('parseAmount', () => {
	it.each([
		['1250', 125000],
		['1250.5', 125050],
		['1250,50', 125050],
		['  0.01 ', 1],
		['1 250.75', 125075]
	])('acepta %j', (input, cents) => {
		expect(parseAmount(input)).toEqual({ cents });
	});

	it.each([
		['', /Ingresa un monto/],
		['abc', /válido/],
		['1.250,50', /válido/], // separador de miles: ambiguo
		['12.345', /válido/], // tres decimales
		['-5', /válido/],
		['0', /mayor que cero/],
		['0.00', /mayor que cero/],
		['1000000000000', /válido/], // 13 dígitos enteros
		['999999999999.99', null]
	])('valida %j', (input, error) => {
		const result = parseAmount(input);
		if (error) expect(result.error).toMatch(error);
		else expect(result.error).toBeUndefined();
	});
});
