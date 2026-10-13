import { describe, expect, it } from 'vitest';
import {
	browserTimezone,
	hasErrors,
	validateEmail,
	validateLogin,
	validateNewPassword,
	validateRegister
} from './validation.js';

describe('validateEmail', () => {
	it.each(['ana@example.com', '  ana.perez+bb@mail.co.cr  '])('acepta %j', (email) => {
		expect(validateEmail(email)).toBeUndefined();
	});

	it.each([
		'',
		'   ',
		'ana',
		'ana@',
		'ana@example',
		'ana @example.com',
		`${'a'.repeat(250)}@x.com`
	])('rechaza %j', (email) => {
		expect(validateEmail(email)).toBeTypeOf('string');
	});
});

describe('validateNewPassword', () => {
	it.each([
		['', 'Ingresa una contraseña.'],
		['corta', /al menos 8/],
		['ñ'.repeat(37), /demasiado larga/] // 37 caracteres pero 74 bytes UTF-8
	])('rechaza %j', (password, want) => {
		expect(validateNewPassword(password)).toMatch(want);
	});

	it.each(['contraseña', 'a'.repeat(72), 'ñ'.repeat(36)])('acepta %j', (password) => {
		expect(validateNewPassword(password)).toBeUndefined();
	});

	it('cuenta caracteres, no bytes, para el mínimo', () => {
		// 8 caracteres multibyte: igual que utf8.RuneCountInString en el backend.
		expect(validateNewPassword('ñññññññ')).toMatch(/al menos 8/);
		expect(validateNewPassword('ññññññññ')).toBeUndefined();
	});
});

describe('validateLogin', () => {
	it('no exige longitud mínima de contraseña al iniciar sesión', () => {
		expect(validateLogin({ email: 'ana@example.com', password: 'x' })).toEqual({});
	});

	it('reporta los campos vacíos', () => {
		expect(Object.keys(validateLogin({ email: '', password: '' }))).toEqual(['email', 'password']);
	});
});

describe('validateRegister', () => {
	const valid = {
		fullName: 'Ana Pérez',
		email: 'ana@example.com',
		password: 'contraseña-segura',
		currency: 'CRC'
	};

	it('acepta un registro válido', () => {
		expect(hasErrors(validateRegister(valid))).toBe(false);
	});

	it.each([
		['fullName', { fullName: '   ' }],
		['fullName', { fullName: 'a'.repeat(151) }],
		['email', { email: 'ana' }],
		['password', { password: '1234' }],
		['currency', { currency: 'usd' }]
	])('marca %s como inválido', (field, override) => {
		expect(Object.keys(validateRegister({ ...valid, ...override }))).toEqual([field]);
	});
});

describe('browserTimezone', () => {
	it('devuelve una zona horaria IANA', () => {
		expect(browserTimezone()).toMatch(/^[A-Za-z_]+(\/[A-Za-z_+-]+)*$|^UTC$/);
	});
});
