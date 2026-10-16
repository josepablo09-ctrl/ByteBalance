<script>
	/**
	 * Cifra destacada del resumen financiero.
	 *
	 * @typedef {object} Props
	 * @property {string} label
	 * @property {string} value Monto ya formateado.
	 * @property {string} hint
	 * @property {'neutral' | 'positive' | 'negative'} [tone]
	 * @property {boolean} [isLoading]
	 */

	/** @type {Props} */
	let { label, value, hint, tone = 'neutral', isLoading = false } = $props();

	const TONES = {
		neutral: 'text-zinc-50',
		positive: 'text-emerald-400',
		negative: 'text-red-400'
	};
</script>

<div class="rounded-lg border border-zinc-800 bg-zinc-900/40 px-5 py-4">
	<p class="text-sm text-zinc-400">{label}</p>
	{#if isLoading}
		<div class="mt-2.5 h-7 w-36 animate-pulse rounded-md bg-zinc-800/70" aria-hidden="true"></div>
	{:else}
		<p class={['mt-2 text-2xl font-semibold tracking-tight tabular-nums', TONES[tone]]}>{value}</p>
	{/if}
	<p class="mt-1.5 text-xs text-zinc-500">{hint}</p>
</div>
