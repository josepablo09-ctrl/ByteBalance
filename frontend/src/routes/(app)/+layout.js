import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { session } from '#lib/stores/session.js';

// Las pantallas de la app requieren una sesión vigente.
export function load() {
	if (!session.isAuthenticated()) redirect(307, resolve('/(auth)/login'));
}
