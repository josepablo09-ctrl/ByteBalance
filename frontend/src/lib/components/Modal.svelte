<script>
	import Icon from './Icon.svelte';

	/**
	 * Diálogo modal sobre `<dialog>` nativo: atrapa el foco, se cierra con Escape
	 * o al hacer clic fuera, y devuelve el foco al elemento que lo abrió.
	 *
	 * @typedef {object} Props
	 * @property {boolean} [open]
	 * @property {string} title
	 * @property {string} [description]
	 * @property {import('svelte').Snippet} children
	 */

	/** @type {Props} */
	let { open = $bindable(false), title, description, children } = $props();

	const id = $props.id();
	/** @type {HTMLDialogElement | undefined} */
	let dialog = $state();

	$effect(() => {
		if (!dialog) return;
		if (open && !dialog.open) dialog.showModal();
		else if (!open && dialog.open) dialog.close();
	});

	/** @param {MouseEvent} event */
	function handleClick(event) {
		// Un clic directamente sobre el <dialog> cae en el fondo, fuera del panel.
		if (event.target === dialog) open = false;
	}
</script>

<dialog
	bind:this={dialog}
	onclose={() => (open = false)}
	onclick={handleClick}
	aria-labelledby="{id}-title"
	aria-describedby={description ? `${id}-description` : undefined}
	class="m-auto w-[calc(100%-2rem)] max-w-md rounded-lg border border-zinc-800 bg-zinc-950 p-0 text-zinc-100 backdrop:bg-zinc-950/80"
>
	{#if open}
		<div class="flex items-start justify-between gap-4 border-b border-zinc-800 px-5 py-4">
			<div>
				<h2 id="{id}-title" class="text-base font-medium text-zinc-50">{title}</h2>
				{#if description}
					<p id="{id}-description" class="mt-0.5 text-sm text-zinc-400">{description}</p>
				{/if}
			</div>
			<button
				type="button"
				onclick={() => (open = false)}
				aria-label="Cerrar"
				class="-mr-1.5 rounded-md p-1.5 text-zinc-500 transition-colors hover:bg-zinc-900 hover:text-zinc-200 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none"
			>
				<Icon name="close" />
			</button>
		</div>
		<div class="px-5 py-5">
			{@render children()}
		</div>
	{/if}
</dialog>
