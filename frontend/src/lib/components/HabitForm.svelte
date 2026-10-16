<script>
	import { errorMessage } from '#lib/api/errors.js';
	import {
		buildHabitInput,
		describeGoal,
		FREQUENCY_OPTIONS,
		MAX_DESCRIPTION_LENGTH,
		MAX_NAME_LENGTH
	} from '#lib/habits.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import SegmentedControl from './SegmentedControl.svelte';
	import TextField from './TextField.svelte';

	/**
	 * Formulario para crear un hábito o editar uno existente.
	 *
	 * @typedef {object} Props
	 * @property {import('#lib/api/habits.js').Habit} [habit] Si se indica, se edita.
	 * @property {(input: import('#lib/api/habits.js').HabitInput) => Promise<void>} onSubmit
	 *   Guarda el hábito; si falla debe lanzar el error de la API.
	 * @property {() => void} oncancel
	 */

	/** @type {Props} */
	let { habit, onSubmit, oncancel } = $props();

	// Los valores iniciales se toman una vez: después pertenecen al formulario.
	// svelte-ignore state_referenced_locally
	const initial = habit;
	let name = $state(initial?.name ?? '');
	let description = $state(initial?.description ?? '');
	let frequency = $state(
		/** @type {import('#lib/api/habits.js').Frequency} */ (initial?.frequency ?? 'daily')
	);
	let target = $state(String(initial?.target_count ?? 1));
	let hasReminder = $state(Boolean(initial?.reminder_time));
	let reminderTime = $state(initial?.reminder_time ?? '08:00');

	/** @type {Partial<Record<string, string>>} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	const goalPreview = $derived(
		/^\d+$/.test(target) && Number(target) >= 1
			? describeGoal({ frequency, target_count: Number(target) })
			: ''
	);

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		const result = buildHabitInput({
			name,
			description,
			frequency,
			target,
			hasReminder,
			reminderTime
		});
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

	<TextField
		id="habit_name"
		label="Nombre"
		placeholder="Leer 20 páginas"
		maxlength={MAX_NAME_LENGTH}
		bind:value={name}
		error={errors.name}
		oninput={() => (errors.name = undefined)}
	/>

	<TextField
		id="habit_description"
		label="Descripción"
		placeholder="Antes de dormir, sin pantallas"
		maxlength={MAX_DESCRIPTION_LENGTH}
		bind:value={description}
		error={errors.description}
		hint="Opcional."
	/>

	<div class="space-y-1.5">
		<span class="block text-sm font-medium text-zinc-300" aria-hidden="true">Frecuencia</span>
		<SegmentedControl
			label="Frecuencia"
			options={FREQUENCY_OPTIONS}
			value={frequency}
			onchange={(value) =>
				(frequency = /** @type {import('#lib/api/habits.js').Frequency} */ (value))}
			fullWidth
		/>
	</div>

	<TextField
		id="habit_target"
		label="Meta por periodo"
		inputmode="numeric"
		bind:value={target}
		error={errors.target}
		hint={goalPreview ? `Meta: ${goalPreview}.` : undefined}
		oninput={() => (errors.target = undefined)}
	/>

	<div class="space-y-3 rounded-lg border border-zinc-800 bg-zinc-900/40 p-3">
		<div class="flex items-center justify-between gap-4">
			<div>
				<p id="habit-reminder-label" class="text-sm font-medium text-zinc-300">Recordatorio</p>
				<p class="text-xs text-zinc-500">
					Hora del aviso diario, en tu zona horaria. El envío por Telegram aún no está activo.
				</p>
			</div>
			<button
				type="button"
				role="switch"
				aria-checked={hasReminder}
				aria-labelledby="habit-reminder-label"
				onclick={() => (hasReminder = !hasReminder)}
				class={[
					'relative h-5 w-9 shrink-0 rounded-full border transition-colors focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none',
					hasReminder ? 'border-emerald-500/60 bg-emerald-500/30' : 'border-zinc-700 bg-zinc-800'
				]}
			>
				<span
					class={[
						'absolute top-0.5 left-0.5 size-3.5 rounded-full transition-transform',
						hasReminder ? 'translate-x-4 bg-emerald-300' : 'bg-zinc-400'
					]}
				></span>
			</button>
		</div>
		{#if hasReminder}
			<TextField
				id="habit_reminder"
				label="Hora"
				type="time"
				bind:value={reminderTime}
				error={errors.reminderTime}
				oninput={() => (errors.reminderTime = undefined)}
			/>
		{/if}
	</div>

	<div class="flex justify-end gap-2 border-t border-zinc-800 pt-5">
		<Button variant="secondary" onclick={oncancel}>Cancelar</Button>
		<Button type="submit" loading={isSubmitting}>
			{isSubmitting ? 'Guardando…' : habit ? 'Guardar cambios' : 'Crear hábito'}
		</Button>
	</div>
</form>
