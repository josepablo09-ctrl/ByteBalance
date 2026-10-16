<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { currencyOptions } from '#lib/currencies.js';
	import { todayInZone } from '#lib/dates.js';
	import {
		buildTransactionInput,
		MAX_CATEGORY_NAME_LENGTH,
		MAX_DESCRIPTION_LENGTH,
		validateCategoryName
	} from '#lib/transactions.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import SegmentedControl from './SegmentedControl.svelte';
	import SelectField from './SelectField.svelte';
	import TextField from './TextField.svelte';

	/** Valor del selector de categoría que abre la creación de una nueva. */
	const NEW_CATEGORY = '__new__';

	const TYPE_OPTIONS = [
		{ value: 'expense', label: 'Gasto' },
		{ value: 'income', label: 'Ingreso' }
	];

	/**
	 * @typedef {import('#lib/api/finance.js').Category} Category
	 * @typedef {import('#lib/api/finance.js').EntryType} EntryType
	 *
	 * @typedef {object} Props
	 * @property {Category[]} categories
	 * @property {string} defaultCurrency
	 * @property {string} timeZone Zona horaria del usuario.
	 * @property {(input: import('#lib/api/finance.js').RecordTransactionInput) => Promise<void>} onSubmit
	 *   Registra la transacción; si falla debe lanzar el error de la API.
	 * @property {(name: string, type: EntryType) => Promise<Category>} onCreateCategory
	 * @property {() => void} oncancel
	 */

	/** @type {Props} */
	let { categories, defaultCurrency, timeZone, onSubmit, onCreateCategory, oncancel } = $props();

	const today = $derived(todayInZone(timeZone));

	let type = $state(/** @type {EntryType} */ ('expense'));
	let amount = $state('');
	// La moneda inicial es la del usuario; después la elige el formulario.
	// svelte-ignore state_referenced_locally
	let currency = $state(defaultCurrency);
	let categoryId = $state('');
	let description = $state('');
	// svelte-ignore state_referenced_locally
	let date = $state(todayInZone(timeZone));

	/** @type {Partial<Record<string, string>>} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	let isCreatingCategory = $state(false);
	let newCategoryName = $state('');
	let categoryError = $state('');
	let isSavingCategory = $state(false);

	const categoryOptions = $derived([
		{ value: '', label: 'Sin categoría' },
		...categories.filter((c) => c.type === type).map((c) => ({ value: c.id, label: c.name })),
		{ value: NEW_CATEGORY, label: '+ Nueva categoría…' }
	]);

	/** @param {string} next */
	function handleTypeChange(next) {
		type = /** @type {EntryType} */ (next);
		// Una categoría de gastos no aplica a un ingreso, y viceversa.
		if (!categories.some((c) => c.id === categoryId && c.type === type)) categoryId = '';
		isCreatingCategory = false;
	}

	/** @param {string} value */
	function handleCategoryChange(value) {
		if (value !== NEW_CATEGORY) return;
		categoryId = '';
		isCreatingCategory = true;
		newCategoryName = '';
		categoryError = '';
	}

	async function createCategory() {
		categoryError = validateCategoryName(newCategoryName) ?? '';
		if (categoryError) return;
		isSavingCategory = true;
		try {
			const category = await onCreateCategory(newCategoryName.trim(), type);
			categoryId = category.id;
			isCreatingCategory = false;
		} catch (err) {
			categoryError = errorMessage(err);
		} finally {
			isSavingCategory = false;
		}
	}

	/** @param {KeyboardEvent} event */
	function handleCategoryKeydown(event) {
		// Enter crea la categoría en lugar de enviar el formulario completo.
		if (event.key === 'Enter') {
			event.preventDefault();
			createCategory();
		}
	}

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		const result = buildTransactionInput(
			{ type, amount, currency, categoryId, description, date },
			{ timeZone, today }
		);
		if (result.errors) {
			errors = result.errors;
			return;
		}
		errors = {};
		isSubmitting = true;
		try {
			await onSubmit(result.input);
		} catch (err) {
			formError = errorMessage(err);
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} novalidate class="space-y-5">
	{#if formError}
		<Alert message={formError} />
	{/if}

	<SegmentedControl
		label="Tipo de movimiento"
		options={TYPE_OPTIONS}
		value={type}
		onchange={handleTypeChange}
		fullWidth
	/>

	<div class="grid grid-cols-[minmax(0,1fr)_7.5rem] gap-3">
		<TextField
			id="amount"
			label="Monto"
			inputmode="decimal"
			placeholder="0.00"
			bind:value={amount}
			error={errors.amount}
			oninput={() => (errors.amount = undefined)}
		/>
		<SelectField
			id="currency"
			label="Moneda"
			options={currencyOptions(defaultCurrency).map((c) => ({ value: c.value, label: c.value }))}
			bind:value={currency}
			error={errors.currency}
		/>
	</div>

	{#if isCreatingCategory}
		<div class="space-y-2 rounded-lg border border-zinc-800 bg-zinc-900/40 p-3">
			<TextField
				id="new_category"
				label={type === 'income' ? 'Nueva categoría de ingresos' : 'Nueva categoría de gastos'}
				placeholder={type === 'income' ? 'Salario' : 'Supermercado'}
				maxlength={MAX_CATEGORY_NAME_LENGTH}
				bind:value={newCategoryName}
				error={categoryError}
				oninput={() => (categoryError = '')}
				onkeydown={handleCategoryKeydown}
			/>
			<div class="flex justify-end gap-2">
				<Button variant="ghost" class="h-8 px-3" onclick={() => (isCreatingCategory = false)}>
					Cancelar
				</Button>
				<Button
					variant="secondary"
					class="h-8 px-3"
					loading={isSavingCategory}
					onclick={createCategory}
				>
					Crear categoría
				</Button>
			</div>
		</div>
	{:else}
		<SelectField
			id="category"
			label="Categoría"
			options={categoryOptions}
			bind:value={categoryId}
			onchange={handleCategoryChange}
		/>
	{/if}

	<TextField
		id="description"
		label="Descripción"
		placeholder={type === 'income' ? 'Pago de octubre' : 'Compra semanal'}
		maxlength={MAX_DESCRIPTION_LENGTH}
		bind:value={description}
		error={errors.description}
		hint="Opcional."
	/>

	<TextField
		id="date"
		label="Fecha"
		type="date"
		bind:value={date}
		error={errors.date}
		oninput={() => (errors.date = undefined)}
	/>

	<div class="flex justify-end gap-2 border-t border-zinc-800 pt-5">
		<Button variant="secondary" onclick={oncancel}>Cancelar</Button>
		<Button type="submit" loading={isSubmitting}>
			{isSubmitting ? 'Guardando…' : type === 'income' ? 'Registrar ingreso' : 'Registrar gasto'}
		</Button>
	</div>
</form>
