<script>
	import { onMount } from 'svelte';
	import { errorMessage, financeApi } from '#lib/api/index.js';
	import { monthLabel, monthRange } from '#lib/dates.js';
	import { formatMoney } from '#lib/money.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import AvailableBreakdown from './AvailableBreakdown.svelte';
	import SavingsCard from './SavingsCard.svelte';
	import SummaryCard from './SummaryCard.svelte';

	/**
	 * Tarjetas de balance total, ingresos, gastos y ahorro del mes, y el desglose
	 * del disponible real para gastar, en la moneda principal del usuario. Los
	 * montos en otras monedas se listan aparte: no se suman porque no hay tipo de
	 * cambio.
	 *
	 * @typedef {object} Props
	 * @property {string} currency Moneda principal del usuario.
	 * @property {string} timeZone
	 * @property {number | null} savingsGoalCents Meta de ahorro mensual, en `currency`.
	 * @property {(goalCents: number | null) => Promise<void>} onSaveGoal
	 */

	/** @type {Props} */
	let { currency, timeZone, savingsGoalCents, onSaveGoal } = $props();

	/** @type {import('#lib/api/finance.js').Balance[]} */
	let allTime = $state([]);
	/** @type {import('#lib/api/finance.js').Balance[]} */
	let thisMonth = $state([]);
	let isLoading = $state(true);
	let loadError = $state('');
	let requestId = 0;

	/** @param {import('#lib/api/finance.js').Balance[]} balances */
	const inCurrency = (balances) =>
		balances.find((b) => b.currency === currency) ?? {
			currency,
			income_cents: 0,
			expense_cents: 0,
			net_cents: 0,
			saved_cents: 0,
			withdrawn_cents: 0,
			savings_cents: 0,
			available_cents: 0
		};

	const total = $derived(inCurrency(allTime));
	const month = $derived(inCurrency(thisMonth));
	const otherCurrencies = $derived(allTime.filter((b) => b.currency !== currency));

	/** Vuelve a pedir el balance (p. ej. tras registrar una transacción). */
	export async function refresh() {
		const current = ++requestId;
		isLoading = true;
		loadError = '';
		try {
			const [all, monthly] = await Promise.all([
				financeApi.balance(),
				financeApi.balance(monthRange(timeZone))
			]);
			if (current !== requestId) return;
			allTime = all;
			thisMonth = monthly;
		} catch (err) {
			if (current === requestId) loadError = errorMessage(err);
		} finally {
			if (current === requestId) isLoading = false;
		}
	}

	onMount(refresh);
</script>

<section aria-label="Resumen" class="space-y-3">
	{#if loadError}
		<div class="flex flex-wrap items-center gap-3">
			<Alert message={loadError} />
			<Button variant="secondary" class="h-9" onclick={refresh}>Reintentar</Button>
		</div>
	{/if}

	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
		<SummaryCard
			label="Balance total"
			value={formatMoney(total.net_cents, currency)}
			hint={`Histórico en ${currency}`}
			tone={total.net_cents < 0 ? 'negative' : 'neutral'}
			{isLoading}
		/>
		<SummaryCard
			label="Ingresos del mes"
			value={formatMoney(month.income_cents, currency)}
			hint={monthLabel(timeZone)}
			tone="positive"
			{isLoading}
		/>
		<SummaryCard
			label="Gastos del mes"
			value={formatMoney(month.expense_cents, currency)}
			hint={monthLabel(timeZone)}
			{isLoading}
		/>
		<SavingsCard
			monthCents={month.savings_cents}
			totalCents={total.savings_cents}
			{currency}
			monthLabel={monthLabel(timeZone)}
			goalCents={savingsGoalCents}
			{isLoading}
			{onSaveGoal}
		/>
	</div>

	<AvailableBreakdown
		netCents={total.net_cents}
		savingsCents={total.savings_cents}
		availableCents={total.available_cents}
		{currency}
		{isLoading}
	/>

	{#if !isLoading && otherCurrencies.length > 0}
		<p class="text-xs text-zinc-500">
			Otras monedas (balance histórico):
			{#each otherCurrencies as balance, i (balance.currency)}
				<span class="text-zinc-300 tabular-nums"
					>{formatMoney(balance.net_cents, balance.currency)}</span
				>{i < otherCurrencies.length - 1 ? ' · ' : ''}
			{/each}
		</p>
	{/if}
</section>
