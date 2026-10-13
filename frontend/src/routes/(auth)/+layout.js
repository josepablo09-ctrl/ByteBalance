import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { session } from '#lib/stores/session.js';

// Quien ya tiene sesión no necesita ver login ni registro.
export function load() {
	if (session.isAuthenticated()) redirect(307, resolve('/(app)'));
}
