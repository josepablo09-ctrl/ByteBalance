<script>
	import { onMount } from 'svelte';
	import { errorMessage } from '#lib/api/errors.js';
	import { formatGeneratedTime, formatPeriod, paragraphs } from '#lib/insight.js';
	import { lastInsight } from '#lib/stores/insight.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import Icon from './Icon.svelte';
	import Spinner from './Spinner.svelte';

	/**
	 * Análisis inteligente: un diagnóstico con IA de las finanzas y la constancia
	 * de hábitos del usuario. El último análisis se conserva durante la sesión.
	 *
	 * @typedef {object} Props
	 * @property {Pick<import('#lib/api/ai.js').AIApi, 'status' | 'diagnose'>} ai
	 * @property {string} timeZone
	 * @property {number} [slowAfterMs] A partir de cuándo se avisa que está tardando.
	 */

	/** @type {Props} */
	let { ai, timeZone, slowAfterMs = 8000 } = $props();

	let isEnabled = $state(/** @type {boolean | null} */ (null));
	let statusError = $state('');
	let isAnalyzing = $state(false);
	let isSlow = $state(false);
	let error = $state('');

	const diagnosis = $derived($lastInsight);

	async function loadStatus() {
		statusError = '';
		try {
			isEnabled = (await ai.status()).enabled;
		} catch (err) {
			statusError = errorMessage(err);
		}
	}

	async function analyze() {
		isAnalyzing = true;
		isSlow = false;
		error = '';
		const slowTimer = setTimeout(() => (isSlow = true), slowAfterMs);
		try {
			lastInsight.set(await ai.diagnose());
		} catch (err) {
			error = errorMessage(err);
		} finally {
			clearTimeout(slowTimer);
			isAnalyzing = false;
		}
	}

	onMount(loadStatus);
</script>

{#snippet insightList(/** @type {string} */ title, /** @type {string[]} */ items)}
	<div class="rounded-lg border border-zinc-800 bg-zinc-950/40 p-4">
		<h3 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">{title}</h3>
		<ul class="mt-3 space-y-2.5">
			{#each items as item, i (i)}
				<li class="flex gap-2.5 text-sm leading-relaxed text-zinc-300">
					<span class="mt-2 size-1.5 shrink-0 rounded-full bg-emerald-400/70" aria-hidden="true"
					></span>
					{item}
				</li>
			{/each}
		</ul>
	</div>
{/snippet}

<section
	aria-labelledby="ai-insight-title"
	class="rounded-lg border border-zinc-800 bg-zinc-900/30"
>
	<div class="flex flex-wrap items-start gap-3 border-b border-zinc-800 px-5 py-4">
		<span
			class="grid size-9 shrink-0 place-items-center rounded-lg border border-emerald-500/20 bg-emerald-500/10 text-emerald-300"
			aria-hidden="true"
		>
			<Icon name="sparkles" />
		</span>
		<div class="min-w-0 flex-1 basis-56">
			<h2 id="ai-insight-title" class="text-sm font-medium text-zinc-100">Análisis inteligente</h2>
			<p class="mt-0.5 text-sm text-zinc-400">
				Un diagnóstico de tus finanzas y tus hábitos de los últimos 30 días, con sugerencias
				personalizadas.
			</p>
		</div>
		{#if diagnosis && !isAnalyzing && isEnabled}
			<Button variant="secondary" class="h-9" onclick={analyze}>
				<Icon name="sparkles" />
				Actualizar análisis
			</Button>
		{/if}
	</div>

	<div class="space-y-4 px-5 py-5">
		{#if error}
			<div class="flex flex-wrap items-center gap-3">
				<Alert message={error} />
			</div>
		{/if}

		{#if isEnabled === null && !statusError}
			<div class="h-10 w-44 animate-pulse rounded-lg bg-zinc-800/70" aria-label="Cargando"></div>
		{:else if statusError}
			<div class="flex flex-wrap items-center gap-3">
				<Alert message={statusError} />
				<Button variant="secondary" class="h-9" onclick={loadStatus}>Reintentar</Button>
			</div>
		{:else if !isEnabled}
			<p class="text-sm text-zinc-500">
				El análisis inteligente aún no está disponible en este servidor.
			</p>
		{:else if isAnalyzing}
			<div class="space-y-5" aria-busy="true">
				<p class="flex items-center gap-2 text-sm text-zinc-400" role="status">
					<Spinner class="size-4 text-emerald-400" />
					{isSlow
						? 'Está tardando más de lo habitual: el servicio de IA está muy solicitado…'
						: 'Analizando tus finanzas y hábitos…'}
				</p>
				<div class="animate-pulse space-y-5" aria-hidden="true">
					<div class="space-y-2.5">
						<div class="h-5 w-3/4 rounded-md bg-zinc-800/80"></div>
						<div class="h-3.5 w-full rounded bg-zinc-800/60"></div>
						<div class="h-3.5 w-11/12 rounded bg-zinc-800/60"></div>
						<div class="h-3.5 w-2/3 rounded bg-zinc-800/60"></div>
					</div>
					<div class="grid gap-3 md:grid-cols-2">
						<div class="h-28 rounded-lg border border-zinc-800 bg-zinc-900/60"></div>
						<div class="h-28 rounded-lg border border-zinc-800 bg-zinc-900/60"></div>
					</div>
					<div class="grid gap-3 sm:grid-cols-2">
						<div class="h-20 rounded-lg bg-zinc-800/40"></div>
						<div class="h-20 rounded-lg bg-zinc-800/40"></div>
					</div>
				</div>
			</div>
		{:else if diagnosis}
			<article class="space-y-6">
				<div class="space-y-3">
					<p class="text-lg leading-snug font-semibold text-zinc-50">{diagnosis.headline}</p>
					{#each paragraphs(diagnosis.summary) as paragraph, i (i)}
						<p class="text-sm leading-relaxed text-zinc-400">{paragraph}</p>
					{/each}
				</div>

				{#if diagnosis.finance_insights.length || diagnosis.habit_insights.length}
					<div class="grid gap-3 md:grid-cols-2">
						{#if diagnosis.finance_insights.length}
							{@render insightList('Finanzas', diagnosis.finance_insights)}
						{/if}
						{#if diagnosis.habit_insights.length}
							{@render insightList('Hábitos', diagnosis.habit_insights)}
						{/if}
					</div>
				{/if}

				{#if diagnosis.suggestions.length}
					<div>
						<h3 class="text-xs font-medium tracking-wide text-zinc-500 uppercase">Sugerencias</h3>
						<ol class="mt-3 grid gap-3 sm:grid-cols-2">
							{#each diagnosis.suggestions as suggestion, i (i)}
								<li class="flex gap-3 rounded-lg border border-zinc-800 bg-zinc-950/40 p-4">
									<span
										class="grid size-6 shrink-0 place-items-center rounded-md border border-emerald-500/25 font-mono text-xs text-emerald-300"
									>
										{i + 1}
									</span>
									<div class="min-w-0">
										<p class="text-sm font-medium text-zinc-100">{suggestion.title}</p>
										<p class="mt-1 text-sm leading-relaxed text-zinc-400">{suggestion.detail}</p>
									</div>
								</li>
							{/each}
						</ol>
					</div>
				{/if}

				<p class="border-t border-zinc-800 pt-4 text-xs text-zinc-500">
					Periodo {formatPeriod(diagnosis.period)} · Generado a las
					{formatGeneratedTime(diagnosis.generated_at, timeZone)} · Generado con IA: puede contener errores,
					no es asesoría financiera profesional.
				</p>
			</article>
		{:else}
			<div class="flex flex-wrap items-center justify-between gap-4">
				<p class="max-w-prose text-sm text-zinc-400">
					La IA revisa tus ingresos, gastos y la constancia de tus hábitos. Solo recibe totales y
					nombres de categorías y hábitos, nunca tu nombre, email ni las descripciones de tus
					movimientos.
				</p>
				<Button onclick={analyze}>
					<Icon name="sparkles" />
					Generar análisis
				</Button>
			</div>
		{/if}
	</div>
</section>
