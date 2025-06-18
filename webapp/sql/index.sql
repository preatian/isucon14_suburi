-- ride_statuses テーブル
ALTER TABLE ride_statuses ADD INDEX idx_ride_id_created_at (ride_id, created_at DESC);

-- chairs テーブル
ALTER TABLE chairs ADD INDEX idx_owner_id (owner_id);
ALTER TABLE chairs ADD UNIQUE INDEX uq_access_token (access_token);
-- ALTER TABLE chairs ADD PRIMARY KEY (id); -- If 'id' is not already the primary key

-- rides テーブル
ALTER TABLE rides ADD INDEX idx_chair_id_updated_at (chair_id, updated_at DESC);
ALTER TABLE rides ADD INDEX idx_user_id_created_at (user_id, created_at DESC);
-- ALTER TABLE rides ADD PRIMARY KEY (id); -- If 'id' is not already the primary key

-- chair_locations テーブル
ALTER TABLE chair_locations ADD INDEX idx_chair_id_created_at (chair_id, created_at DESC);
-- ALTER TABLE chair_locations ADD PRIMARY KEY (id); -- If 'id' is not already the primary key

-- coupons テーブル
ALTER TABLE coupons ADD INDEX idx_user_id_used_by_created_at (user_id, used_by, created_at);
ALTER TABLE coupons ADD INDEX idx_user_id_code_used_by (user_id, code, used_by);
ALTER TABLE coupons ADD INDEX idx_used_by (used_by);

-- users テーブル
-- ALTER TABLE users ADD PRIMARY KEY (id); -- If 'id' is not already the primary key
ALTER TABLE users ADD UNIQUE INDEX uq_access_token (access_token);

-- owners テーブル
ALTER TABLE owners ADD UNIQUE INDEX uq_access_token (access_token);
ALTER TABLE owners ADD UNIQUE INDEX uq_chair_register_token (chair_register_token);
-- ALTER TABLE owners ADD PRIMARY KEY (id); -- If 'id' is not already the primary key

-- payment_tokens テーブル
ALTER TABLE payment_tokens ADD INDEX idx_user_id (user_id);

-- settings テーブル
ALTER TABLE settings ADD UNIQUE INDEX uq_name (name);