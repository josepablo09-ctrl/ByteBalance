-- Revierte 000001_init_schema en orden inverso de dependencias.
DROP TABLE IF EXISTS habit_logs;
DROP TABLE IF EXISTS habits;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS set_updated_at();
