-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    username varchar(30) NOT NULL UNIQUE,
    password varchar(200) NOT NULL,

    PRIMARY KEY (id)
);

CREATE TABLE user_private_keys (
    user_id UUID NOT NULL,
    private_key_id UUID NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(private_key_id) REFERENCES private_keys(id) ON DELETE CASCADE,
    UNIQUE(user_id, private_key_id),
    UNIQUE(private_key_id)
);

CREATE TABLE user_refresh_tokens (
    user_id UUID NOT NULL,
    refresh_token_id UUID NOT NULL,

    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(refresh_token_id) REFERENCES refresh_tokens(id) ON DELETE CASCADE,
    UNIQUE(user_id, refresh_token_id),
    UNIQUE(refresh_token_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS user_private_keys CASCADE;
DROP TABLE IF EXISTS user_refresh_tokens CASCADE;
-- +goose StatementEnd
