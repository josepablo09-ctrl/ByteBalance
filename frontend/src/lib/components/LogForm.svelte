<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { buildLogInput, countOn, MAX_NOTE_LENGTH } from '#lib/habits.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import TextField from './TextField.svelte';

	/**
	 * Registra, corrige o borra cuántas veces se cumplió un hábito en un día. El
	 * backend guarda un registro por día: el nuevo valor reemplaza al anterior, y
	 * 0 lo borra.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/habits.js').HabitLog[]} logs Registros ya cargados, para precargar el día.
	 * @property {string} today `YYYY-MM-DD` en la zona del usuario.
	 * @property {(action: Exclude<import('#lib/habits.js').LogAction, { errors: object }>) => Promise<void>} onSubmit
	 *   Guarda (`input`) o borra (`remove`) el día; si falla debe lanzar el error de la API.
	 * @property {() => void} oncancel
	 */

	/** @type {Props} */
	let { logs, today, onSubmit, oncancel } = $props();

	// svelte-ignore state_referenced_locally
	let date = $state(today);
	let count = $state('1');
	let note = $state('');
	/** @type {Partial<Record<string, string>>} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	const existing = $derived(countOn(logs, date));
	const isRemoving = $derived(count.trim() === '0' && existing > 0);

	/** Propone lo ya registrado en el día elegido, para poder corregirlo. */
	function prefill() {
		const found = logs.find((l) => l.date === date);
		count = String(found?.count ?? 1);
		note = found?.note ?? '';
	}
	prefill();

	function handleDateInput() {
		errors.date = undefined;
		prefill();
	}

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		const result = buildLogInput({ date, count, note }, { today, existingCount: existing });
		if (result.errors) {
			errors = result.errors;
			return;
		}
		errors = {};
		isSubmitting = true;
		try {
			await onSubmit(result);
		} catch (err) {
			formError = errorMessage(err);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} novalidate class="space-y-5">
	{#if formError}
		<Alert message={formError} />
	{/if}

	<div class="grid grid-cols-[minmax(0,1fr)_7rem] gap-3">
		<TextField
			id="log_date"
			label="Día"
			type="date"
			bind:value={date}
			error={errors.date}
			oninput={handleDateInput}
		/>
		<TextField
			id="log_count"
			label="Veces"
			inputmode="numeric"
			bind:value={count}
			error={errors.count}
			oninput={() => (errors.count = undefined)}
		/>
	</div>

	{#if existing > 0}
		<p class="text-xs text-zinc-500">
			Ese día ya tiene {existing === 1 ? '1 vez registrada' : `${existing} veces registradas`}; lo
			que guardes lo reemplaza. Usa 0 para borrarlo.
		</p>
	{/if}

	<TextField
		id="log_note"
		label="Nota"
		placeholder="¿Cómo te fue?"
		maxlength={MAX_NOTE_LENGTH}
		bind:value={note}
		error={errors.note}
		hint="Opcional."
	/>

	<div class="flex justify-end gap-2 border-t border-zinc-800 pt-5">
		<Button variant="secondary" onclick={oncancel}>Cancelar</Button>
		<Button type="submit" variant={isRemoving ? 'danger' : 'primary'} loading={isSubmitting}>
			{isSubmitting ? 'Guardando…' : isRemoving ? 'Borrar registro' : 'Guardar'}
		</Button>
	</div>
</form>
