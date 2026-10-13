<script>
	import FormField, { CONTROL_CLASS, borderClass, messageId } from './FormField.svelte';

	/**
	 * @typedef {object} Props
	 * @property {string} id
	 * @property {string} label
	 * @property {string} [value]
	 * @property {'text' | 'email' | 'password'} [type]
	 * @property {string} [error]
	 * @property {string} [hint]
	 * @property {import('svelte/elements').FullAutoFill} [autocomplete]
	 * @property {string} [placeholder]
	 * @property {boolean} [disabled]
	 * @property {() => void} [oninput]
	 * @property {import('svelte').Snippet} [trailing] Contenido a la derecha del input (p. ej. un botón).
	 */

	/** @type {Props} */
	let {
		id,
		label,
		value = $bindable(''),
		type = 'text',
		error,
		hint,
		autocomplete,
		placeholder,
		disabled = false,
		oninput,
		trailing
	} = $props();
</script>

<FormField {id} {label} {error} {hint}>
	<div class="relative">
		<input
			{id}
			name={id}
			{type}
			{autocomplete}
			{placeholder}
			{disabled}
			{oninput}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={error || hint ? messageId(id) : undefined}
			class={[CONTROL_CLASS, borderClass(error), trailing && 'pr-20']}
		/>
		{#if trailing}
			<div class="absolute inset-y-0 right-0 flex items-center pr-1.5">
				{@render trailing()}
			</div>
		{/if}
	</div>
</FormField>
