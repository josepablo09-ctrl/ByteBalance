<script>
	import FormField, { CONTROL_CLASS, borderClass, messageId } from './FormField.svelte';

	/**
	 * @typedef {object} Props
	 * @property {string} id
	 * @property {string} label
	 * @property {string} [value]
	 * @property {'text' | 'email' | 'password' | 'date'} [type]
	 * @property {'text' | 'decimal'} [inputmode] Teclado sugerido en móviles.
	 * @property {number} [maxlength]
	 * @property {string} [error]
	 * @property {string} [hint]
	 * @property {import('svelte/elements').FullAutoFill} [autocomplete]
	 * @property {string} [placeholder]
	 * @property {boolean} [disabled]
	 * @property {() => void} [oninput]
	 * @property {(event: KeyboardEvent) => void} [onkeydown]
	 * @property {import('svelte').Snippet} [trailing] Contenido a la derecha del input (p. ej. un botón).
	 */

	/** @type {Props} */
	let {
		id,
		label,
		value = $bindable(''),
		type = 'text',
		inputmode,
		maxlength,
		error,
		hint,
		autocomplete,
		placeholder,
		disabled = false,
		oninput,
		onkeydown,
		trailing
	} = $props();
</script>

<FormField {id} {label} {error} {hint}>
	<div class="relative">
		<input
			{id}
			name={id}
			{type}
			{inputmode}
			{maxlength}
			{autocomplete}
			{placeholder}
			{disabled}
			{oninput}
			{onkeydown}
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
