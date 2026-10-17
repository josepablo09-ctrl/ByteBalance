/**
 * Tipo de movimiento. `saving` aparta dinero del disponible al ahorro y
 * `withdrawal` lo devuelve; ninguno cambia el balance total.
 *
 * @typedef {'income' | 'expense' | 'saving' | 'withdrawal'} EntryType
 */

/** @typedef {'income' | 'expense'} CategoryType Las categorías son solo de ingresos o gastos. */

/**
 * @typedef {object} Category
 * @property {string} id
 * @property {string} name
 * @property {CategoryType} type
 * @property {string} created_at
 */

/**
 * @typedef {object} Transaction
 * @property {string} id
 * @property {EntryType} type
 * @property {number} amount_cents Siempre positivo; el tipo indica el signo.
 * @property {string} currency
 * @property {string | null} category_id
 * @property {string} description
 * @property {string} occurred_at Fecha RFC 3339.
 * @property {string} created_at
 */

/**
 * @typedef {object} Balance
 * @property {string} currency
 * @property {number} income_cents
 * @property {number} expense_cents
 * @property {number} net_cents Balance total: ingresos menos gastos.
 * @property {number} saved_cents Apartado al ahorro.
 * @property {number} withdrawn_cents Retirado del ahorro.
 * @property {number} savings_cents Ahorro neto: apartado menos retirado.
 * @property {number} available_cents Disponible real para gastar: `net_cents - savings_cents`.
 */

/**
 * @typedef {object} RecordTransactionInput
 * @property {EntryType} type
 * @property {number} amount_cents
 * @property {string} [currency] Vacía = moneda por defecto del usuario.
 * @property {string | null} [category_id]
 * @property {string} [description]
 * @property {string} [occurred_at] RFC 3339; ausente = ahora.
 */

/**
 * @typedef {object} TransactionFilter
 * @property {Date} [from] Inicio inclusivo.
 * @property {Date} [to] Fin exclusivo.
 * @property {EntryType} [type]
 * @property {string} [categoryId]
 * @property {number} [limit] Máximo 200 (el backend usa 50 por defecto).
 * @property {number} [offset]
 */

/** @param {Date | undefined} date */
const toIso = (date) => date?.toISOString();

/**
 * Endpoints de finanzas: categorías, transacciones y balance.
 *
 * @param {import('./client.js').ApiClient} client
 */
export function createFinanceApi(client) {
	return {
		/** @returns {Promise<Category[]>} */
		async listCategories() {
			const { categories } = await client.get('/categories');
			return categories;
		},

		/**
		 * @param {string} name
		 * @param {CategoryType} type
		 * @returns {Promise<Category>}
		 */
		createCategory: (name, type) => client.post('/categories', { name, type }),

		/**
		 * Transacciones más recientes primero.
		 *
		 * @param {TransactionFilter} [filter]
		 * @returns {Promise<Transaction[]>}
		 */
		async listTransactions({ from, to, type, categoryId, limit, offset } = {}) {
			const { transactions } = await client.get('/transactions', {
				query: { from: toIso(from), to: toIso(to), type, category_id: categoryId, limit, offset }
			});
			return transactions;
		},

		/**
		 * @param {RecordTransactionInput} input
		 * @returns {Promise<Transaction>}
		 */
		recordTransaction: (input) => client.post('/transactions', input),

		/**
		 * Corrige una transacción; solo cambian los campos enviados y
		 * `category_id: null` quita la categoría.
		 *
		 * @param {string} id
		 * @param {Partial<RecordTransactionInput>} changes
		 * @returns {Promise<Transaction>}
		 */
		updateTransaction: (id, changes) =>
			client.patch(`/transactions/${encodeURIComponent(id)}`, changes),

		/**
		 * @param {string} id
		 * @returns {Promise<void>}
		 */
		async deleteTransaction(id) {
			await client.delete(`/transactions/${encodeURIComponent(id)}`);
		},

		/**
		 * Ingresos, gastos y neto por moneda en [from, to). Sin fechas, de toda la historia.
		 *
		 * @param {{ from?: Date, to?: Date }} [range]
		 * @returns {Promise<Balance[]>}
		 */
		async balance({ from, to } = {}) {
			const { balances } = await client.get('/balance', {
				query: { from: toIso(from), to: toIso(to) }
			});
			return balances;
		}
	};
}

/** @typedef {ReturnType<typeof createFinanceApi>} FinanceApi */
