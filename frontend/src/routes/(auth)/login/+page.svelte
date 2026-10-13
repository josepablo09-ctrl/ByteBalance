<script>
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { authApi } from '#lib/api/index.js';
	import AuthLayout from '#lib/components/AuthLayout.svelte';
	import LoginForm from '#lib/components/LoginForm.svelte';
	import { session } from '#lib/stores/session.js';

	const notice = $derived(
		page.url.searchParams.has('registered')
			? 'Tu cuenta está lista. Inicia sesión para continuar.'
			: undefined
	);

	/** @param {{ email: string, password: string }} credentials */
	async function handleLogin({ email, password }) {
		const token = await authApi.login(email, password);
		session.start({ token: token.access_token, expiresAt: token.expires_at });
		await goto(resolve('/(app)'), { replaceState: true });
	}
</script>

<svelte:head>
	<title>Iniciar sesión · ByteBalance</title>
</svelte:head>

<AuthLayout title="Inicia sesión" subtitle="Accede a tu balance de hábitos y finanzas.">
	<LoginForm onLogin={handleLogin} {notice} />

	{#snippet footer()}
		¿No tienes cuenta?
		<a
			href={resolve('/(auth)/register')}
			class="font-medium text-emerald-400 transition-colors hover:text-emerald-300"
		>
			Crea una gratis
		</a>
	{/snippet}
</AuthLayout>
