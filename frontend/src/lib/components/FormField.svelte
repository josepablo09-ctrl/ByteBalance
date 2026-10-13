<script module>
	/** Clases comunes de los controles de formulario (input, select). */
	export const CONTROL_CLASS =
		'block h-10 w-full rounded-lg border bg-zinc-900/60 px-3 text-sm text-zinc-100 transition-colors outline-none placeholder:text-zinc-600 focus:border-emerald-500/70 focus:ring-2 focus:ring-emerald-500/20 disabled:cursor-not-allowed disabled:opacity-60';

	/** @param {string | undefined} error */
	export const borderClass = (error) =>
		error ? 'border-red-500/60' : 'border-zinc-800 hover:border-zinc-700';

	/** Id del párrafo de error o ayuda asociado al control, para `aria-describedby`. */
	export const messageId = (/** @type {string} */ id) => `${id}-message`;
</script>

<script>
	/**
	 * Etiqueta, control y mensaje de error o ayuda de un campo de formulario.
	 *
	 * @typedef {object} Props
	 * @property {string} id Id del control asociado a la etiqueta.
	 * @property {string} label
	 * @property {string} [error] Tiene prioridad sobre `hint`.
	 * @property {string} [hint]
	 * @property {import('svelte').Snippet} children El control.
	 */

	/** @type {Props} */
	let { id, label, error, hint, children } = $props();
</script>

<div class="space-y-1.5">
	<label for={id} class="block text-sm font-medium text-zinc-300">{label}</label>
	{@render children()}
	{#if error}
		<p id={messageId(id)} class="text-xs text-red-400">{error}</p>
	{:else if hint}
		<p id={messageId(id)} class="text-xs text-zinc-500">{hint}</p>
	{/if}
</div>
