<script>
	import { ApiError } from '#lib/api/client.js';
	import { errorMessage } from '#lib/api/errors.js';
	import {
		browserTimezone,
		hasErrors,
		MIN_PASSWORD_LENGTH,
		validateRegister
	} from '#lib/validation.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import PasswordField from './PasswordField.svelte';
	import SelectField from './SelectField.svelte';
	import TextField from './TextField.svelte';

	const CURRENCIES = [
		{ value: 'USD', label: 'USD · Dólar estadounidense' },
		{ value: 'EUR', label: 'EUR · Euro' },
		{ value: 'CRC', label: 'CRC · Colón costarricense' },
		{ value: 'MXN', label: 'MXN · Peso mexicano' },
		{ value: 'COP', label: 'COP · Peso colombiano' },
		{ value: 'ARS', label: 'ARS · Peso argentino' },
		{ value: 'CLP', label: 'CLP · Peso chileno' },
		{ value: 'PEN', label: 'PEN · Sol peruano' },
		{ value: 'GBP', label: 'GBP · Libra esterlina' }
	];

	/**
	 * @typedef {object} Props
	 * @property {(input: import('#lib/api/auth.js').RegisterInput) => Promise<void>} onRegister
	 *   Crea la cuenta; si falla debe lanzar el error de la API.
	 */

	/** @type {Props} */
	let { onRegister } = $props();

	const timezone = browserTimezone();

	let fullName = $state('');
	let email = $state('');
	let password = $state('');
	let currency = $state('USD');
	/** @type {import('#lib/validation.js').FieldErrors} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		errors = validateRegister({ fullName, email, password, currency });
		if (hasErrors(errors)) return;

		isSubmitting = true;
		try {
			await onRegister({
				full_name: fullName.trim(),
				email: email.trim(),
				password,
				default_currency: currency,
				timezone
			});
		} catch (err) {
			// El email duplicado se muestra junto al campo que hay que corregir.
			if (err instanceof ApiError && err.code === 'email_taken') {
				errors = { email: errorMessage(err) };
			} else {
				formError = errorMessage(err);
			}
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} novalidate class="space-y-5">
	{#if formError}
		<Alert message={formError} />
	{/if}

	<TextField
		id="full_name"
		label="Nombre completo"
		autocomplete="name"
		placeholder="Ana Pérez"
		bind:value={fullName}
		error={errors.fullName}
		oninput={() => (errors.fullName = undefined)}
	/>
	<TextField
		id="email"
		label="Email"
		type="email"
		autocomplete="email"
		placeholder="tu@email.com"
		bind:value={email}
		error={errors.email}
		oninput={() => (errors.email = undefined)}
	/>
	<PasswordField
		id="password"
		label="Contraseña"
		autocomplete="new-password"
		hint={`Mínimo ${MIN_PASSWORD_LENGTH} caracteres. Una frase larga es más segura.`}
		bind:value={password}
		error={errors.password}
		oninput={() => (errors.password = undefined)}
	/>
	<SelectField
		id="default_currency"
		label="Moneda principal"
		options={CURRENCIES}
		bind:value={currency}
		error={errors.currency}
		hint="Podrás registrar transacciones en otras monedas."
	/>

	<p class="text-xs text-zinc-500">
		Zona horaria: <span class="font-mono text-zinc-400">{timezone}</span>
		<span class="text-zinc-600">· detectada en tu navegador</span>
	</p>

	<Button type="submit" class="w-full" loading={isSubmitting}>
		{isSubmitting ? 'Creando cuenta…' : 'Crear cuenta'}
	</Button>
</form>
