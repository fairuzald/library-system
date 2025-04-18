-- migrate:up
CREATE TABLE book_categories (
    book_id UUID NOT NULL,
    category_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (book_id, category_id)
);

-- Add comments to describe the purpose of the table
COMMENT ON TABLE book_categories IS 'Junction table for many-to-many relationship between books and categories';
COMMENT ON COLUMN book_categories.book_id IS 'Foreign key to books table';
COMMENT ON COLUMN book_categories.category_id IS 'Foreign key to categories table';

CREATE INDEX idx_book_categories_book_id ON book_categories(book_id);
CREATE INDEX idx_book_categories_category_id ON book_categories(category_id);

-- migrate:down
DROP TABLE IF EXISTS book_categories;
