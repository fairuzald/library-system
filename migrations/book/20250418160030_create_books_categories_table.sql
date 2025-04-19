-- migrate:up
CREATE TABLE IF NOT EXISTS books_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL,
    category_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT books_categories_book_id_category_id_key UNIQUE (book_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_books_categories_book_id ON books_categories(book_id);
CREATE INDEX IF NOT EXISTS idx_books_categories_category_id ON books_categories(category_id);

-- migrate:down
DROP TABLE IF EXISTS books_categories;
