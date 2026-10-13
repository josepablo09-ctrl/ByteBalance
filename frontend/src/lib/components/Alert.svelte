<script>
	/**
	 * @typedef {object} Props
	 * @property {string} message
	 * @property {'error' | 'success'} [tone]
	 */

	/** @type {Props} */
	let { message, tone = 'error' } = $props();
</script>

<div
	role={tone === 'error' ? 'alert' : 'status'}
	class={[
		'flex items-start gap-2.5 rounded-lg border px-3 py-2.5 text-sm',
		tone === 'error'
			? 'border-red-500/25 bg-red-500/10 text-red-300'
			: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-300'
	]}
>
	<svg
		class="mt-0.5 size-4 shrink-0"
		viewBox="0 0 16 16"
		fill="none"
		stroke="currentColor"
		stroke-width="1.5"
		aria-hidden="true"
	>
		<circle cx="8" cy="8" r="6.25" />
		{#if tone === 'error'}
			<path d="M8 4.75v3.75M8 10.9v.1" stroke-linecap="round" />
		{:else}
			<path d="m5.5 8.25 1.75 1.75 3.25-3.5" stroke-linecap="round" stroke-linejoin="round" />
		{/if}
	</svg>
	<p>{message}</p>
</div>
