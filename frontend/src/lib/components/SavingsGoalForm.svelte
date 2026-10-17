<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { centsToDecimal, parseAmount } from '#lib/money.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import TextField from './TextField.svelte';

	/**
	 * Fija o quita la meta de ahorro mensual.
	 *
	 * @typedef {object} Props
	 * @property {number | null} goalCents Meta actual; `null` si no hay.
	 * @property {string} currency Moneda de la meta (la principal del usuario).
	 * @property {(goalCents: number | null) => Promise<void>} onSubmit Si falla debe lanzar el error de la API.
	 * @property {() => void} oncancel
	 */

	/** @type {Props} */
	let { goalCents, currency, onSubmit, oncancel } = $props();

	// svelte-ignore state_referenced_locally
	let amount = $state(goalCents ? centsToDecimal(goalCents) : '');
	let amountError = $state('');
	let formError = $state('');
	let busy = $state(/** @type {'' | 'save' | 'remove'} */ (''));

	/** @param {SubmitEvent} event */
	async function save(event) {
		event.preventDefault();
		const parsed = parseAmount(amount);
		if (parsed.error || parsed.cents === undefined) {
			amountError = parsed.error ?? 'Ingresa un monto.';
			return;
		}
		await submit('save', parsed.cents);
	}

	/**
	 * @param {'save' | 'remove'} action
	 * @param {number | null} cents
	 */
	async function submit(action, cents) {
		busy = action;
		formError = '';
		try {
			await onSubmit(cents);
		} catch (err) {
			formError = errorMessage(err);
		} finally {
			busy = '';
		}
	}
</script>

<form onsubmit={save} novalidate class="space-y-5">
	{#if formError}
		<Alert message={formError} />
	{/if}

	<TextField
		id="savings_goal"
		label={`Meta mensual (${currency})`}
		inputmode="decimal"
		placeholder="100000"
		bind:value={amount}
		error={amountError}
		oninput={() => (amountError = '')}
		hint="Cuánto quieres apartar cada mes. Se compara con lo que apartas menos lo que retiras."
	/>

	<div class="flex flex-wrap justify-between gap-2 border-t border-zinc-800 pt-5">
		{#if goalCents}
			<Button
				variant="ghost"
				class="hover:text-red-300"
				loading={busy === 'remove'}
				disabled={busy !== ''}
				onclick={() => submit('remove', null)}
			>
				Quitar meta
			</Button>
		{:else}
			<span></span>
		{/if}
		<div class="flex gap-2">
			<Button variant="secondary" onclick={oncancel}>Cancelar</Button>
			<Button type="submit" loading={busy === 'save'} disabled={busy !== ''}>Guardar meta</Button>
		</div>
	</div>
</form>
