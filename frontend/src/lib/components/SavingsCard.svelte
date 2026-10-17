<script>
	import { goalProgress } from '#lib/entryTypes.js';
	import { formatMoney } from '#lib/money.js';
	import Icon from './Icon.svelte';
	import Modal from './Modal.svelte';
	import SavingsGoalForm from './SavingsGoalForm.svelte';

	/**
	 * Ahorro del mes frente a la meta mensual, con el ahorro acumulado.
	 *
	 * @typedef {object} Props
	 * @property {number} monthCents Apartado menos retirado en el mes.
	 * @property {number} totalCents Ahorro acumulado.
	 * @property {string} currency
	 * @property {string} monthLabel P. ej. "octubre de 2026".
	 * @property {number | null} goalCents
	 * @property {boolean} [isLoading]
	 * @property {(goalCents: number | null) => Promise<void>} onSaveGoal
	 */

	/** @type {Props} */
	let {
		monthCents,
		totalCents,
		currency,
		monthLabel,
		goalCents,
		isLoading = false,
		onSaveGoal
	} = $props();

	let isEditing = $state(false);

	const progress = $derived(goalProgress(monthCents, goalCents));

	/** @param {number | null} cents */
	async function saveGoal(cents) {
		await onSaveGoal(cents);
		isEditing = false;
	}
</script>

<div class="rounded-lg border border-zinc-800 bg-zinc-900/40 px-5 py-4">
	<div class="flex items-center justify-between gap-2">
		<p class="text-sm text-zinc-400">Ahorro del mes</p>
		<button
			type="button"
			class="-mr-1.5 flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs text-zinc-500 transition-colors hover:bg-zinc-800/70 hover:text-zinc-200 focus-visible:ring-2 focus-visible:ring-emerald-500/40 focus-visible:outline-none"
			onclick={() => (isEditing = true)}
		>
			<Icon name="edit" class="size-3.5" />
			{goalCents ? 'Editar meta' : 'Fijar meta'}
		</button>
	</div>

	{#if isLoading}
		<div class="mt-2.5 h-7 w-36 animate-pulse rounded-md bg-zinc-800/70" aria-hidden="true"></div>
		<div
			class="mt-3 h-1.5 w-full animate-pulse rounded-full bg-zinc-800/70"
			aria-hidden="true"
		></div>
	{:else}
		<p class="mt-2 text-2xl font-semibold tracking-tight text-zinc-50 tabular-nums">
			{formatMoney(monthCents, currency)}
		</p>
		{#if progress !== null && goalCents}
			<div
				class="mt-3 h-1.5 overflow-hidden rounded-full bg-zinc-800"
				role="progressbar"
				aria-label="Progreso de la meta de ahorro"
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={progress}
			>
				<div
					class="h-full rounded-full bg-emerald-400 transition-[width]"
					style:width={`${progress}%`}
				></div>
			</div>
			<p class="mt-1.5 text-xs text-zinc-500">
				<span class={progress === 100 ? 'text-emerald-400' : 'text-zinc-300'}>{progress} %</span>
				de {formatMoney(goalCents, currency)} · Acumulado {formatMoney(totalCents, currency)}
			</p>
		{:else}
			<p class="mt-1.5 text-xs text-zinc-500">
				{monthLabel} · Acumulado {formatMoney(totalCents, currency)}
			</p>
		{/if}
	{/if}
</div>

<Modal
	bind:open={isEditing}
	title="Meta de ahorro mensual"
	description="Fija cuánto quieres apartar cada mes para ver tu progreso."
>
	{#if isEditing}
		<SavingsGoalForm
			{goalCents}
			{currency}
			onSubmit={saveGoal}
			oncancel={() => (isEditing = false)}
		/>
	{/if}
</Modal>
