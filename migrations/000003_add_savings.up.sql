-- 000003_add_savings: movimientos de ahorro y meta mensual de ahorro.
--
-- 'saving' aparta dinero del disponible para gastar y 'withdrawal' lo devuelve.
-- Ninguno cambia el balance total (ingresos - gastos): el dinero sigue siendo
-- del usuario, solo cambia si está disponible o apartado.

ALTER TABLE transactions DROP CONSTRAINT transactions_type_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_type_check
    CHECK (type IN ('income', 'expense', 'saving', 'withdrawal'));

-- Las categorías son solo de ingresos o gastos: el ahorro no se categoriza.
ALTER TABLE transactions ADD CONSTRAINT transactions_saving_without_category
    CHECK (category_id IS NULL OR type IN ('income', 'expense'));

-- Meta de ahorro mensual en la moneda por defecto del usuario; NULL = sin meta.
ALTER TABLE users ADD COLUMN monthly_savings_goal NUMERIC(14,2);
ALTER TABLE users ADD CONSTRAINT users_monthly_savings_goal_positive
    CHECK (monthly_savings_goal IS NULL OR monthly_savings_goal > 0);
