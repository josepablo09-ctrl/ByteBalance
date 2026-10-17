import { describe, expect, it } from 'vitest';
import { formatGeneratedTime, formatPeriod, paragraphs } from './insight.js';

describe('formatPeriod', () => {
	it('muestra las fechas de calendario sin correrlas por la zona horaria', () => {
		const label = formatPeriod({ from: '2026-09-17', to: '2026-10-16' });
		expect(label).toMatch(/^17 sept? – 16 oct$/);
	});
});

describe('formatGeneratedTime', () => {
	it('usa la zona horaria del usuario', () => {
		expect(formatGeneratedTime('2026-10-16T20:30:00Z', 'America/Costa_Rica')).toBe('14:30');
		expect(formatGeneratedTime('2026-10-16T20:30:00Z', 'UTC')).toBe('20:30');
	});
});

describe('paragraphs', () => {
	it('separa por líneas en blanco y descarta vacíos', () => {
		expect(paragraphs('Uno.\n\nDos.\n \n\nTres.')).toEqual(['Uno.', 'Dos.', 'Tres.']);
		expect(paragraphs('  ')).toEqual([]);
	});
});
