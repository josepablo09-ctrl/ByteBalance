<script>
	/** @type {{ class?: string }} */
	let { class: className = '' } = $props();
</script>

<span class={['inline-flex items-center gap-2.5', className]}>
	<span
		class="grid size-7 place-items-center rounded-md border border-zinc-800 bg-zinc-900"
		aria-hidden="true"
	>
		<svg viewBox="0 0 16 16" class="size-4" fill="none">
			<rect x="2" y="8" width="3" height="6" rx="0.75" class="fill-zinc-500" />
			<rect x="6.5" y="5" width="3" height="9" rx="0.75" class="fill-emerald-400" />
			<rect x="11" y="2" width="3" height="12" rx="0.75" class="fill-zinc-300" />
		</svg>
	</span>
	<span class="text-[15px] font-semibold tracking-tight text-zinc-100">ByteBalance</span>
</span>
