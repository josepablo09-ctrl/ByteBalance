<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { buildLogInput, countOn, MAX_NOTE_LENGTH } from '#lib/habits.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import TextField from './TextField.svelte';

	/**
	 * Registra (o corrige) cuántas veces se cumplió un hábito en un día. El
	 * backend guarda un registro por día: el nuevo valor reemplaza al anterior.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/habits.js').HabitLog[]} logs Registros ya cargados, para precargar el día.
	 * @property {string} today `YYYY-MM-DD` en la zona del usuario.
	 * @property {(input: import('#lib/api/habits.js').LogInput) => Promise<void>} onSubmit
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

	/** Al elegir otro día, propone lo ya registrado para poder corregirlo. */
	function handleDateInput() {
		errors.date = undefined;
		const found = logs.find((l) => l.date === date);
		count = String(found?.count ?? 1);
		note = found?.note ?? '';
	}

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		const result = buildLogInput({ date, count, note }, today);
		if (result.errors) {
			errors = result.errors;
			return;
		}
		errors = {};
		isSubmitting = true;
		try {
			await onSubmit(result.input);
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
			que guardes lo reemplaza.
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
		<Button type="submit" loading={isSubmitting}>{isSubmitting ? 'Guardando…' : 'Guardar'}</Button>
	</div>
</form>
