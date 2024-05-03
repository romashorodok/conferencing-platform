-- +goose Up
-- +goose StatementBegin
CREATE TABLE private_keys (
    id UUID NOT NULL DEFAULT uuid_generate_v4(),
    jws_message json NOT NULL,

    PRIMARY KEY(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS private_keys CASCADE;
-- +goose StatementEnd
