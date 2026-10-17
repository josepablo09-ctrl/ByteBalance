/**
 * Presentación de cada tipo de movimiento. `availableSign` es su efecto en el
 * disponible para gastar: los ingresos y los retiros del ahorro lo aumentan; los
 * gastos y los aportes al ahorro lo reducen.
 *
 * @typedef {object} EntryTypeInfo
 * @property {string} label Nombre corto (selector de tipo).
 * @property {string} title Título de una transacción sin descripción ni categoría.
 * @property {string} srPrefix Texto para lectores de pantalla antes del monto.
 * @property {string} submitLabel Botón del formulario al registrar.
 * @property {string} notice Aviso tras registrar.
 * @property {string} placeholder Ejemplo de descripción.
 * @property {1 | -1} availableSign
 * @property {boolean} categorizable Si admite categorías (solo ingresos y gastos).
 * @property {'income' | 'expense' | 'piggy' | 'withdraw'} icon
 * @property {string} tone Clase de color del monto y el ícono.
 */

/** @type {Record<import('./api/finance.js').EntryType, EntryTypeInfo>} */
export const ENTRY_TYPES = {
	expense: {
		label: 'Gasto',
		title: 'Gasto',
		srPrefix: 'Gasto de',
		submitLabel: 'Registrar gasto',
		notice: 'Gasto registrado.',
		placeholder: 'Compra semanal',
		availableSign: -1,
		categorizable: true,
		icon: 'expense',
		tone: 'text-zinc-200'
	},
	income: {
		label: 'Ingreso',
		title: 'Ingreso',
		srPrefix: 'Ingreso de',
		submitLabel: 'Registrar ingreso',
		notice: 'Ingreso registrado.',
		placeholder: 'Pago de octubre',
		availableSign: 1,
		categorizable: true,
		icon: 'income',
		tone: 'text-emerald-400'
	},
	saving: {
		label: 'Ahorro',
		title: 'Aporte al ahorro',
		srPrefix: 'Aporte al ahorro de',
		submitLabel: 'Apartar al ahorro',
		notice: 'Dinero apartado al ahorro.',
		placeholder: 'Fondo de emergencia',
		availableSign: -1,
		categorizable: false,
		icon: 'piggy',
		tone: 'text-zinc-300'
	},
	withdrawal: {
		label: 'Retiro',
		title: 'Retiro del ahorro',
		srPrefix: 'Retiro del ahorro de',
		submitLabel: 'Retirar del ahorro',
		notice: 'Retiro del ahorro registrado.',
		placeholder: 'Reparación del auto',
		availableSign: 1,
		categorizable: false,
		icon: 'withdraw',
		tone: 'text-zinc-300'
	}
};

/**
 * Efecto de una transacción en el disponible para gastar, en centavos con signo.
 *
 * @param {Pick<import('./api/finance.js').Transaction, 'type' | 'amount_cents'>} transaction
 */
export function availableEffect({ type, amount_cents }) {
	return ENTRY_TYPES[type].availableSign * amount_cents;
}

/**
 * Progreso hacia la meta de ahorro mensual, entre 0 y 100 (redondeado hacia
 * abajo para no mostrar 100 % antes de cumplirla). `null` si no hay meta.
 *
 * @param {number} savedCents Ahorro neto del mes; puede ser negativo.
 * @param {number | null | undefined} goalCents
 */
export function goalProgress(savedCents, goalCents) {
	if (!goalCents || goalCents <= 0) return null;
	return Math.min(100, Math.max(0, Math.floor((savedCents / goalCents) * 100)));
}
