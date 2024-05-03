-- +goose Up
-- +goose StatementBegin
CREATE TABLE refresh_tokens (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    private_key_id UUID NOT NULL,
    plaintext text NOT NULL,

    created_at TIMESTAMPTZ(6) NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ(6) NOT NULL,

    PRIMARY KEY(id),
    FOREIGN KEY(private_key_id) REFERENCES private_keys(id),
    UNIQUE(private_key_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_tokens CASCADE;
-- +goose StatementEnd
