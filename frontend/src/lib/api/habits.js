/** @typedef {'daily' | 'weekly' | 'monthly'} Frequency */

/**
 * @typedef {object} Habit
 * @property {string} id
 * @property {string} name
 * @property {string} description
 * @property {Frequency} frequency
 * @property {number} target_count Veces por periodo.
 * @property {string | null} reminder_time `HH:MM` en la zona del usuario.
 * @property {boolean} is_active `false` = pausado.
 * @property {string} created_at
 */

/**
 * @typedef {object} HabitLog
 * @property {string} id
 * @property {string} habit_id
 * @property {string} date `YYYY-MM-DD` en la zona del usuario.
 * @property {number} count
 * @property {string} note
 */

/**
 * @typedef {object} HabitInput Cuerpo para crear o editar un hábito.
 * @property {string} name
 * @property {string} description
 * @property {Frequency} frequency
 * @property {number} target_count
 * @property {string | null} reminder_time
 */

/**
 * @typedef {object} LogInput
 * @property {string} [date] `YYYY-MM-DD`; ausente = hoy para el usuario.
 * @property {number} [count]
 * @property {string} [note]
 */

/**
 * Endpoints de hábitos y sus registros.
 *
 * @param {import('./client.js').ApiClient} client
 */
export function createHabitsApi(client) {
	return {
		/**
		 * @param {{ activeOnly?: boolean }} [options]
		 * @returns {Promise<Habit[]>}
		 */
		async listHabits({ activeOnly = false } = {}) {
			const { habits } = await client.get('/habits', {
				query: { active: activeOnly || undefined }
			});
			return habits;
		},

		/**
		 * El backend ignora `reminder_time: null` al crear (equivale a sin recordatorio).
		 *
		 * @param {HabitInput} input
		 * @returns {Promise<Habit>}
		 */
		createHabit: (input) => client.post('/habits', input),

		/**
		 * Cambios parciales; `is_active` pausa o reanuda.
		 *
		 * @param {string} id
		 * @param {Partial<HabitInput> & { is_active?: boolean }} changes
		 * @returns {Promise<Habit>}
		 */
		updateHabit: (id, changes) => client.patch(`/habits/${encodeURIComponent(id)}`, changes),

		/**
		 * @param {string} id
		 * @returns {Promise<void>}
		 */
		async deleteHabit(id) {
			await client.delete(`/habits/${encodeURIComponent(id)}`);
		},

		/**
		 * Registros con fecha en [from, to], ordenados por fecha.
		 *
		 * @param {string} id
		 * @param {{ from?: string, to?: string }} [range] Fechas `YYYY-MM-DD`.
		 * @returns {Promise<HabitLog[]>}
		 */
		async listLogs(id, { from, to } = {}) {
			const { logs } = await client.get(`/habits/${encodeURIComponent(id)}/logs`, {
				query: { from, to }
			});
			return logs;
		},

		/**
		 * Registra (o reemplaza) el cumplimiento de un día.
		 *
		 * @param {string} id
		 * @param {LogInput} [input]
		 * @returns {Promise<HabitLog>}
		 */
		logCompletion: (id, input = {}) => client.post(`/habits/${encodeURIComponent(id)}/logs`, input),

		/**
		 * Borra el registro de un día (lo deja en cero).
		 *
		 * @param {string} id
		 * @param {string} date `YYYY-MM-DD`
		 * @returns {Promise<void>}
		 */
		async deleteLog(id, date) {
			await client.delete(`/habits/${encodeURIComponent(id)}/logs/${encodeURIComponent(date)}`);
		}
	};
}

/** @typedef {ReturnType<typeof createHabitsApi>} HabitsApi */
