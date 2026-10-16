<script>
	import Spinner from './Spinner.svelte';

	const VARIANTS = {
		primary: 'bg-emerald-500 text-zinc-950 hover:bg-emerald-400 focus-visible:ring-emerald-500/50',
		secondary:
			'border border-zinc-800 bg-zinc-900 text-zinc-200 hover:border-zinc-700 hover:bg-zinc-800/70 focus-visible:ring-zinc-500/40',
		ghost: 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-100 focus-visible:ring-zinc-500/40',
		danger:
			'border border-red-500/30 bg-red-500/10 text-red-300 hover:border-red-500/50 hover:bg-red-500/15 focus-visible:ring-red-500/40'
	};

	/**
	 * @typedef {object} Props
	 * @property {'button' | 'submit'} [type]
	 * @property {keyof typeof VARIANTS} [variant]
	 * @property {boolean} [loading] Muestra un spinner y deshabilita el botón.
	 * @property {boolean} [disabled]
	 * @property {string} [class]
	 * @property {(event: MouseEvent) => void} [onclick]
	 * @property {import('svelte').Snippet} children
	 */

	/** @type {Props} */
	let {
		type = 'button',
		variant = 'primary',
		loading = false,
		disabled = false,
		class: className = '',
		onclick,
		children
	} = $props();
</script>

<button
	{type}
	{onclick}
	disabled={disabled || loading}
	aria-busy={loading || undefined}
	class={[
		'inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-medium transition-colors',
		'focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950 focus-visible:outline-none',
		'disabled:cursor-not-allowed disabled:opacity-60',
		VARIANTS[variant],
		className
	]}
>
	{#if loading}
		<Spinner />
	{/if}
	{@render children()}
</button>
