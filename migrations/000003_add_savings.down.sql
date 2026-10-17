-- Revierte 000003_add_savings. Los movimientos de ahorro no caben en el esquema
-- anterior, así que se eliminan: revertir esta migración pierde esos datos.

ALTER TABLE users DROP CONSTRAINT users_monthly_savings_goal_positive;
ALTER TABLE users DROP COLUMN monthly_savings_goal;

ALTER TABLE transactions DROP CONSTRAINT transactions_saving_without_category;

DELETE FROM transactions WHERE type IN ('saving', 'withdrawal');
ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('income', 'expense'));
