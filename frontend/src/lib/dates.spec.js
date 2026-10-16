import { describe, expect, it } from 'vitest';
import {
	formatShortDate,
	monthLabel,
	monthRange,
	todayInZone,
	zonedParts,
	zonedToInstant
} from './dates.js';

// 2026-10-16 03:30 UTC: en Costa Rica (UTC-6) todavía es 15 de octubre.
const NOW = new Date('2026-10-16T03:30:00Z');

describe('zonedParts', () => {
	it('devuelve la hora de pared de la zona', () => {
		expect(zonedParts(NOW, 'America/Costa_Rica')).toEqual({
			year: 2026,
			month: 10,
			day: 15,
			hour: 21,
			minute: 30,
			second: 0
		});
		expect(zonedParts(NOW, 'Asia/Tokyo')).toMatchObject({ day: 16, hour: 12 });
	});
});

describe('zonedToInstant', () => {
	it.each([
		['America/Costa_Rica', { year: 2026, month: 10, day: 1 }, '2026-10-01T06:00:00.000Z'],
		['Asia/Tokyo', { year: 2026, month: 10, day: 1 }, '2026-09-30T15:00:00.000Z'],
		['UTC', { year: 2026, month: 10, day: 1, hour: 12 }, '2026-10-01T12:00:00.000Z'],
		// Nueva York: horario de verano (UTC-4) en octubre y estándar (UTC-5) en diciembre.
		['America/New_York', { year: 2026, month: 10, day: 1 }, '2026-10-01T04:00:00.000Z'],
		['America/New_York', { year: 2026, month: 12, day: 1 }, '2026-12-01T05:00:00.000Z'],
		// El mes 13 es enero del año siguiente.
		['America/Costa_Rica', { year: 2026, month: 13, day: 1 }, '2027-01-01T06:00:00.000Z']
	])('%s %o', (timeZone, wall, want) => {
		expect(zonedToInstant(wall, timeZone).toISOString()).toBe(want);
	});

	it('resuelve el día del cambio de horario', () => {
		// 2026-11-01 en Nueva York: la medianoche todavía es horario de verano.
		expect(
			zonedToInstant({ year: 2026, month: 11, day: 1 }, 'America/New_York').toISOString()
		).toBe('2026-11-01T04:00:00.000Z');
		// 2026-11-02 ya es horario estándar.
		expect(
			zonedToInstant({ year: 2026, month: 11, day: 2 }, 'America/New_York').toISOString()
		).toBe('2026-11-02T05:00:00.000Z');
	});
});

describe('monthRange', () => {
	it('usa el mes de la zona del usuario, no el de UTC', () => {
		// En UTC ya es 1 de noviembre, pero en Costa Rica aún es 31 de octubre.
		const { from, to } = monthRange('America/Costa_Rica', new Date('2026-11-01T03:00:00Z'));
		expect(from.toISOString()).toBe('2026-10-01T06:00:00.000Z');
		expect(to.toISOString()).toBe('2026-11-01T06:00:00.000Z');
	});

	it('desplaza al mes anterior, cruzando el año', () => {
		const { from, to } = monthRange('UTC', new Date('2026-01-10T00:00:00Z'), -1);
		expect(from.toISOString()).toBe('2025-12-01T00:00:00.000Z');
		expect(to.toISOString()).toBe('2026-01-01T00:00:00.000Z');
	});
});

describe('todayInZone', () => {
	it('formatea la fecha de la zona como valor de input date', () => {
		expect(todayInZone('America/Costa_Rica', NOW)).toBe('2026-10-15');
		expect(todayInZone('Asia/Tokyo', NOW)).toBe('2026-10-16');
	});
});

describe('monthLabel', () => {
	it('nombra el mes con mayúscula inicial', () => {
		expect(monthLabel('America/Costa_Rica', NOW)).toBe('Octubre de 2026');
	});
});

describe('formatShortDate', () => {
	it('muestra día y mes en la zona del usuario', () => {
		expect(formatShortDate('2026-10-16T03:30:00Z', 'America/Costa_Rica', NOW)).toBe('15 oct');
	});

	it('incluye el año si no es el actual', () => {
		expect(formatShortDate('2025-12-24T18:00:00Z', 'UTC', NOW)).toBe('24 dic 2025');
	});
});
