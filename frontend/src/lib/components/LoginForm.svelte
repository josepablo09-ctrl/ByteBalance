<script>
	import { errorMessage } from '#lib/api/errors.js';
	import { hasErrors, validateLogin } from '#lib/validation.js';
	import Alert from './Alert.svelte';
	import Button from './Button.svelte';
	import PasswordField from './PasswordField.svelte';
	import TextField from './TextField.svelte';

	/**
	 * @typedef {object} Props
	 * @property {(credentials: { email: string, password: string }) => Promise<void>} onLogin
	 *   Inicia la sesión; si falla debe lanzar el error de la API.
	 * @property {string} [notice] Mensaje informativo (p. ej. tras registrarse).
	 */

	/** @type {Props} */
	let { onLogin, notice } = $props();

	let email = $state('');
	let password = $state('');
	/** @type {import('#lib/validation.js').FieldErrors} */
	let errors = $state({});
	let formError = $state('');
	let isSubmitting = $state(false);

	/** @param {SubmitEvent} event */
	async function handleSubmit(event) {
		event.preventDefault();
		formError = '';
		errors = validateLogin({ email, password });
		if (hasErrors(errors)) return;

		isSubmitting = true;
		try {
			await onLogin({ email: email.trim(), password });
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
	{:else if notice}
		<Alert message={notice} tone="success" />
	{/if}

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
		autocomplete="current-password"
		bind:value={password}
		error={errors.password}
		oninput={() => (errors.password = undefined)}
	/>

	<Button type="submit" class="w-full" loading={isSubmitting}>
		{isSubmitting ? 'Ingresando…' : 'Iniciar sesión'}
	</Button>
</form>
