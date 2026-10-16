<script>
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { MediaQuery } from 'svelte/reactivity';
	import Icon from './Icon.svelte';
	import Logo from './Logo.svelte';

	/**
	 * Navegación principal. En pantallas grandes es una columna fija; en móvil,
	 * un panel que se abre desde el header.
	 *
	 * @typedef {object} Props
	 * @property {boolean} [open] Visible en móvil.
	 * @property {() => void} [onclose]
	 */

	/** @type {Props} */
	let { open = false, onclose } = $props();

	// Mismo punto de corte que `lg:` de Tailwind.
	const isDesktop = new MediaQuery('min-width: 1024px');

	/** @type {{ label: string, icon: 'wallet' | 'habits' | 'settings', route: '/(app)' | '/(app)/habits' | '/(app)/settings' }[]} */
	const NAV_ITEMS = [
		{ label: 'Finanzas', icon: 'wallet', route: '/(app)' },
		{ label: 'Hábitos', icon: 'habits', route: '/(app)/habits' },
		{ label: 'Ajustes', icon: 'settings', route: '/(app)/settings' }
	];
</script>

{#if open}
	<button
		type="button"
		class="fixed inset-0 z-30 bg-zinc-950/70 lg:hidden"
		aria-label="Cerrar menú"
		onclick={onclose}
	></button>
{/if}

<!-- Cerrado en móvil queda fuera de pantalla: inert evita que reciba el foco con Tab. -->
<aside
	id="app-sidebar"
	inert={!open && !isDesktop.current}
	class={[
		'fixed inset-y-0 left-0 z-40 flex w-60 flex-col border-r border-zinc-800 bg-zinc-950 transition-transform lg:sticky lg:top-0 lg:h-dvh lg:translate-x-0',
		open ? 'translate-x-0' : '-translate-x-full'
	]}
>
	<div class="flex h-14 shrink-0 items-center border-b border-zinc-800 px-5">
		<a href={resolve('/(app)')} aria-label="Inicio" onclick={onclose}>
			<Logo />
		</a>
	</div>

	<nav aria-label="Principal" class="flex-1 space-y-0.5 p-3">
		<p class="px-2 pt-2 pb-1.5 font-mono text-[11px] tracking-wider text-zinc-600 uppercase">
			Espacio
		</p>
		{#each NAV_ITEMS as item (item.label)}
			{@const isActive = page.url.pathname === resolve(item.route)}
			<a
				href={resolve(item.route)}
				onclick={onclose}
				aria-current={isActive ? 'page' : undefined}
				class={[
					'flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-sm transition-colors',
					isActive
						? 'bg-zinc-900 text-zinc-50'
						: 'text-zinc-400 hover:bg-zinc-900/60 hover:text-zinc-200'
				]}
			>
				<Icon name={item.icon} class={isActive ? 'size-4 text-emerald-400' : 'size-4'} />
				{item.label}
			</a>
		{/each}
	</nav>
</aside>
