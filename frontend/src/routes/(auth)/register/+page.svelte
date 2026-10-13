<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { authApi } from '#lib/api/index.js';
	import AuthLayout from '#lib/components/AuthLayout.svelte';
	import RegisterForm from '#lib/components/RegisterForm.svelte';
	import { session } from '#lib/stores/session.js';

	/** @param {import('#lib/api/auth.js').RegisterInput} input */
	async function handleRegister(input) {
		await authApi.register(input);
		// La cuenta ya existe: si el login automático falla (p. ej. por el límite de
		// intentos), se envía al login en lugar de mostrar un error de registro.
		try {
			const token = await authApi.login(input.email, input.password);
			session.start({ token: token.access_token, expiresAt: token.expires_at });
			await goto(resolve('/(app)'), { replaceState: true });
		} catch {
			await goto(resolve('/(auth)/login?registered'), { replaceState: true });
		}
	}
</script>

<svelte:head>
	<title>Crear cuenta · ByteBalance</title>
</svelte:head>

<AuthLayout title="Crea tu cuenta" subtitle="Empieza a registrar tus hábitos y finanzas.">
	<RegisterForm onRegister={handleRegister} />

	{#snippet footer()}
		¿Ya tienes cuenta?
		<a
			href={resolve('/(auth)/login')}
			class="font-medium text-emerald-400 transition-colors hover:text-emerald-300"
		>
			Inicia sesión
		</a>
	{/snippet}
</AuthLayout>
