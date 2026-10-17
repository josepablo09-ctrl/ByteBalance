import { LOCALE } from './locale.js';

/**
 * Formatea el periodo de un análisis (fechas YYYY-MM-DD inclusive), p. ej.
 * "17 sept – 16 oct". Son fechas de calendario: se formatean en UTC para que
 * la zona horaria del navegador no las corra un día.
 *
 * @param {{ from: string, to: string }} period
 */
export function formatPeriod({ from, to }) {
	const format = new Intl.DateTimeFormat(LOCALE, {
		day: 'numeric',
		month: 'short',
		timeZone: 'UTC'
	});
	/** @param {string} day */
	const label = (day) => format.format(new Date(`${day}T00:00:00Z`)).replace('.', '');
	return `${label(from)} – ${label(to)}`;
}

/**
 * Hora local en que se generó un análisis, p. ej. "14:30".
 *
 * @param {string} iso
 * @param {string} timeZone
 */
export function formatGeneratedTime(iso, timeZone) {
	return new Intl.DateTimeFormat(LOCALE, {
		hour: '2-digit',
		minute: '2-digit',
		hourCycle: 'h23',
		timeZone
	}).format(new Date(iso));
}

/**
 * Divide el resumen en párrafos (separados por líneas en blanco).
 *
 * @param {string} text
 * @returns {string[]}
 */
export function paragraphs(text) {
	return text
		.split(/\n\s*\n/)
		.map((p) => p.trim())
		.filter(Boolean);
}
