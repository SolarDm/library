-- +goose Up
CREATE INDEX index_book_id ON book (name);

-- +goose Down
DROP INDEX index_book_id;