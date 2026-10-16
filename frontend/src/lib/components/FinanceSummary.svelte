<script>
	import { onMount } from 'svelte';
	import { errorMessage, financeApi } from '#lib/api/index.js';
	import { monthLabel, monthRange } from '#lib/dates.js';
	import { formatMoney } from '#lib/money.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import SummaryCard from './SummaryCard.svelte';

	/**
	 * Tarjetas de balance total e ingresos y gastos del mes, en la moneda
	 * principal del usuario. Los montos en otras monedas se listan aparte: no se
	 * suman porque no hay tipo de cambio.
	 *
	 * @typedef {object} Props
	 * @property {string} currency Moneda principal del usuario.
	 * @property {string} timeZone
	 */

	/** @type {Props} */
	let { currency, timeZone } = $props();

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
			net_cents: 0
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

	<div class="grid gap-4 sm:grid-cols-3">
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
	</div>

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
