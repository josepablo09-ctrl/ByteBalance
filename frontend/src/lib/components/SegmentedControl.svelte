<script>
	/**
	 * Grupo de botones mutuamente excluyentes (filtros rápidos, tipo de movimiento).
	 *
	 * @typedef {object} Props
	 * @property {string} label Nombre accesible del grupo.
	 * @property {{ value: string, label: string }[]} options
	 * @property {string} [value]
	 * @property {(value: string) => void} [onchange]
	 * @property {boolean} [fullWidth]
	 */

	/** @type {Props} */
	let { label, options, value = $bindable(''), onchange, fullWidth = false } = $props();

	/** @param {string} next */
	function select(next) {
		if (next === value) return;
		value = next;
		onchange?.(next);
	}
</script>

<div
	role="group"
	aria-label={label}
	class={[
		'inline-flex rounded-lg border border-zinc-800 bg-zinc-900/60 p-0.5',
		fullWidth && 'flex w-full'
	]}
>
	{#each options as option (option.value)}
		<button
			type="button"
			aria-pressed={option.value === value}
			onclick={() => select(option.value)}
			class={[
				'h-7 rounded-md px-3 text-xs font-medium transition-colors focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none',
				fullWidth && 'flex-1',
				option.value === value ? 'bg-zinc-800 text-zinc-50' : 'text-zinc-400 hover:text-zinc-200'
			]}
		>
			{option.label}
		</button>
	{/each}
</div>
