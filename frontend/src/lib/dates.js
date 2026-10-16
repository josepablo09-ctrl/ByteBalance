// Fechas en la zona horaria del usuario (perfil), no en la del navegador: así
// "este mes" coincide con lo que calcula el backend aunque viaje o use otro equipo.
import { LOCALE } from './locale.js';

/**
 * @typedef {object} ZonedParts
 * @property {number} year
 * @property {number} month 1-12
 * @property {number} day
 * @property {number} hour
 * @property {number} minute
 * @property {number} second
 */

/** @type {Map<string, Intl.DateTimeFormat>} */
const partsFormatters = new Map();

/**
 * Fecha y hora de pared de `date` en `timeZone`.
 *
 * @param {Date} date
 * @param {string} timeZone Zona IANA, p. ej. `America/Costa_Rica`.
 * @returns {ZonedParts}
 */
export function zonedParts(date, timeZone) {
	let formatter = partsFormatters.get(timeZone);
	if (!formatter) {
		formatter = new Intl.DateTimeFormat('en-US', {
			timeZone,
			hourCycle: 'h23',
			year: 'numeric',
			month: 'numeric',
			day: 'numeric',
			hour: 'numeric',
			minute: 'numeric',
			second: 'numeric'
		});
		partsFormatters.set(timeZone, formatter);
	}
	/** @type {Record<string, number>} */
	const parts = {};
	for (const { type, value } of formatter.formatToParts(date)) {
		if (type !== 'literal') parts[type] = Number(value);
	}
	return /** @type {ZonedParts} */ (/** @type {unknown} */ (parts));
}

/** Diferencia en ms entre la hora de pared en `timeZone` y UTC en el instante `ms`. */
function offsetAt(/** @type {number} */ ms, /** @type {string} */ timeZone) {
	const p = zonedParts(new Date(ms), timeZone);
	return Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second) - (ms - (ms % 1000));
}

/**
 * Instante en que el reloj de `timeZone` marca la fecha y hora indicadas. El
 * mes puede salirse de 1-12 (13 = enero del año siguiente). Si la hora no
 * existe por un cambio de horario, se toma la equivalente después del salto.
 *
 * @param {{ year: number, month: number, day: number, hour?: number, minute?: number }} wall
 * @param {string} timeZone
 */
export function zonedToInstant({ year, month, day, hour = 0, minute = 0 }, timeZone) {
	const asUtc = Date.UTC(year, month - 1, day, hour, minute);
	let ms = asUtc - offsetAt(asUtc, timeZone);
	// El offset puede cambiar entre la estimación y el resultado (horario de verano).
	const corrected = asUtc - offsetAt(ms, timeZone);
	if (corrected !== ms) ms = corrected;
	return new Date(ms);
}

/**
 * Intervalo [from, to) del mes calendario de `now` en `timeZone`, desplazado
 * `monthOffset` meses (-1 = mes anterior).
 *
 * @param {string} timeZone
 * @param {Date} [now]
 * @param {number} [monthOffset]
 * @returns {{ from: Date, to: Date }}
 */
export function monthRange(timeZone, now = new Date(), monthOffset = 0) {
	const { year, month } = zonedParts(now, timeZone);
	return {
		from: zonedToInstant({ year, month: month + monthOffset, day: 1 }, timeZone),
		to: zonedToInstant({ year, month: month + monthOffset + 1, day: 1 }, timeZone)
	};
}

/**
 * Fecha de hoy en `timeZone` con formato `YYYY-MM-DD` (valor de `<input type="date">`).
 *
 * @param {string} timeZone
 * @param {Date} [now]
 */
export function todayInZone(timeZone, now = new Date()) {
	const { year, month, day } = zonedParts(now, timeZone);
	return `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
}

/**
 * Nombre del mes de `now` en `timeZone`, p. ej. "Octubre de 2026".
 *
 * @param {string} timeZone
 * @param {Date} [now]
 */
export function monthLabel(timeZone, now = new Date()) {
	const label = new Intl.DateTimeFormat(LOCALE, {
		month: 'long',
		year: 'numeric',
		timeZone
	}).format(now);
	return label.charAt(0).toUpperCase() + label.slice(1);
}

/**
 * Fecha corta de una transacción en `timeZone`, p. ej. "13 oct"; incluye el año
 * si no es el actual.
 *
 * @param {string} iso Fecha RFC 3339.
 * @param {string} timeZone
 * @param {Date} [now]
 */
export function formatShortDate(iso, timeZone, now = new Date()) {
	const date = new Date(iso);
	const sameYear = zonedParts(date, timeZone).year === zonedParts(now, timeZone).year;
	return new Intl.DateTimeFormat(LOCALE, {
		day: 'numeric',
		month: 'short',
		year: sameYear ? undefined : 'numeric',
		timeZone
	})
		.format(date)
		.replace('.', '');
}
