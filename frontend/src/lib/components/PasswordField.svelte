<script>
	import TextField from './TextField.svelte';

	/** @type {Omit<import('svelte').ComponentProps<typeof TextField>, 'type' | 'trailing'>} */
	let { value = $bindable(''), ...rest } = $props();

	let isVisible = $state(false);
</script>

<TextField {...rest} bind:value type={isVisible ? 'text' : 'password'}>
	{#snippet trailing()}
		<button
			type="button"
			onclick={() => (isVisible = !isVisible)}
			aria-label={isVisible ? 'Ocultar contraseña' : 'Mostrar contraseña'}
			aria-pressed={isVisible}
			class="rounded-md px-2 py-1 text-xs font-medium text-zinc-400 transition-colors hover:text-zinc-100 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none"
		>
			{isVisible ? 'Ocultar' : 'Mostrar'}
		</button>
	{/snippet}
</TextField>
