<script>
	import FormField, { CONTROL_CLASS, borderClass, messageId } from './FormField.svelte';

	/**
	 * @typedef {object} Props
	 * @property {string} id
	 * @property {string} label
	 * @property {string} [value]
	 * @property {{ value: string, label: string }[]} options
	 * @property {string} [error]
	 * @property {string} [hint]
	 * @property {boolean} [disabled]
	 */

	/** @type {Props} */
	let { id, label, value = $bindable(''), options, error, hint, disabled = false } = $props();
</script>

<FormField {id} {label} {error} {hint}>
	<div class="relative">
		<select
			{id}
			name={id}
			{disabled}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={error || hint ? messageId(id) : undefined}
			class={[CONTROL_CLASS, borderClass(error), 'appearance-none pr-9']}
		>
			{#each options as option (option.value)}
				<option value={option.value}>{option.label}</option>
			{/each}
		</select>
		<svg
			class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-zinc-500"
			viewBox="0 0 16 16"
			fill="none"
			stroke="currentColor"
			stroke-width="1.5"
			aria-hidden="true"
		>
			<path d="m4.5 6.5 3.5 3.5 3.5-3.5" stroke-linecap="round" stroke-linejoin="round" />
		</svg>
	</div>
</FormField>
