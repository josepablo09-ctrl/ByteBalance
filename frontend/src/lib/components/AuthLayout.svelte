<script>
	import Logo from './Logo.svelte';

	/**
	 * Estructura de las pantallas de autenticación: panel de producto a la
	 * izquierda (solo en pantallas grandes) y el formulario a la derecha.
	 *
	 * @typedef {object} Props
	 * @property {string} title
	 * @property {string} subtitle
	 * @property {import('svelte').Snippet} children El formulario.
	 * @property {import('svelte').Snippet} [footer] Enlace alternativo (p. ej. a registro).
	 */

	/** @type {Props} */
	let { title, subtitle, children, footer } = $props();

	const FEATURES = [
		{ label: 'Finanzas', text: 'Ingresos y gastos por categoría, en varias monedas.' },
		{ label: 'Hábitos', text: 'Metas diarias, semanales o mensuales con su historial.' },
		{ label: 'Reportes', text: 'Resúmenes automáticos por Telegram e insights con IA.' }
	];
</script>

<div class="grid min-h-dvh lg:grid-cols-2">
	<aside
		class="hidden flex-col justify-between border-r border-zinc-800 bg-zinc-900/30 p-10 lg:flex xl:p-14"
	>
		<Logo />

		<div class="max-w-md">
			<p class="font-mono text-xs tracking-wider text-emerald-400 uppercase">
				Hábitos · Finanzas · Insights
			</p>
			<h2 class="mt-4 text-3xl font-semibold tracking-tight text-balance text-zinc-100">
				Tus hábitos y tu dinero, en un solo balance.
			</h2>
			<p class="mt-4 text-sm leading-relaxed text-zinc-400">
				Registra lo que haces y lo que gastas cada día, y entiende cómo se relacionan.
			</p>

			<dl class="mt-10 divide-y divide-zinc-800 rounded-lg border border-zinc-800 bg-zinc-950/60">
				{#each FEATURES as feature (feature.label)}
					<div class="flex gap-4 px-4 py-3.5">
						<dt class="w-20 shrink-0 font-mono text-xs leading-5 text-zinc-500">{feature.label}</dt>
						<dd class="text-sm leading-5 text-zinc-300">{feature.text}</dd>
					</div>
				{/each}
			</dl>
		</div>

		<p class="text-xs text-zinc-600">© {new Date().getFullYear()} ByteBalance</p>
	</aside>

	<main class="flex flex-col px-6 py-8 sm:px-10">
		<Logo class="lg:hidden" />

		<div class="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center py-12">
			<h1 class="text-2xl font-semibold tracking-tight text-zinc-50">{title}</h1>
			<p class="mt-1.5 text-sm text-zinc-400">{subtitle}</p>

			<div class="mt-8">
				{@render children()}
			</div>

			{#if footer}
				<p class="mt-8 border-t border-zinc-800 pt-6 text-sm text-zinc-400">
					{@render footer()}
				</p>
			{/if}
		</div>
	</main>
</div>
