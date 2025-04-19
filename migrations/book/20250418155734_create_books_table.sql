-- migrate:up
CREATE TABLE IF NOT EXISTS books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) NOT NULL,
    isbn VARCHAR(20) UNIQUE NOT NULL,
    published_year INT NOT NULL,
    publisher VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(50) NOT NULL,
    page_count INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    cover_image TEXT,
    average_rating FLOAT DEFAULT 0,
    quantity INT NOT NULL DEFAULT 1,
    available_quantity INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_author ON books(author);
CREATE INDEX IF NOT EXISTS idx_books_isbn ON books(isbn);
CREATE INDEX IF NOT EXISTS idx_books_status ON books(status);
CREATE INDEX IF NOT EXISTS idx_books_deleted_at ON books(deleted_at);
CREATE INDEX IF NOT EXISTS idx_books_language ON books(language);
CREATE INDEX IF NOT EXISTS idx_books_published_year ON books(published_year);

-- Book Categories relationship table
CREATE TABLE IF NOT EXISTS books_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL,
    category_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT books_categories_book_id_category_id_key UNIQUE (book_id, category_id)
);

-- Add foreign key constraint with cascade delete
ALTER TABLE books_categories
    ADD CONSTRAINT fk_books_categories_book
    FOREIGN KEY (book_id)
    REFERENCES books(id)
    ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_books_categories_book_id ON books_categories(book_id);
CREATE INDEX IF NOT EXISTS idx_books_categories_category_id ON books_categories(category_id);

-- Add some sample books
INSERT INTO books (title, author, isbn, published_year, publisher, description, language, page_count, status)
VALUES
    ('The Great Gatsby', 'F. Scott Fitzgerald', '9780743273565', 1925, 'Scribner', 'A novel about the American Dream', 'English', 180, 'available'),
    ('To Kill a Mockingbird', 'Harper Lee', '9780061120084', 1960, 'HarperCollins', 'Classic of modern American literature', 'English', 281, 'available'),
    ('1984', 'George Orwell', '9780451524935', 1949, 'Signet Classic', 'Dystopian social science fiction', 'English', 328, 'available'),
    ('Pride and Prejudice', 'Jane Austen', '9780141439518', 1813, 'Penguin Classics', 'Romantic novel of manners', 'English', 432, 'available'),
    ('The Hobbit', 'J.R.R. Tolkien', '9780547928227', 1937, 'Houghton Mifflin', 'Fantasy novel', 'English', 310, 'available'),
    ('Harry Potter and the Philosophers Stone', 'J.K. Rowling', '9780747532699', 1997, 'Bloomsbury', 'Fantasy novel', 'English', 223, 'available'),
    ('The Catcher in the Rye', 'J.D. Salinger', '9780316769488', 1951, 'Little, Brown and Company', 'Coming-of-age novel', 'English', 277, 'available'),
    ('Lord of the Flies', 'William Golding', '9780399501487', 1954, 'Perigee Books', 'Allegorical novel', 'English', 224, 'available'),
    ('Animal Farm', 'George Orwell', '9780452284241', 1945, 'Signet Classics', 'Allegorical novella', 'English', 112, 'available'),
    ('Brave New World', 'Aldous Huxley', '9780060850524', 1932, 'Harper Perennial', 'Dystopian novel', 'English', 288, 'available')
ON CONFLICT (isbn) DO NOTHING;

-- migrate:down
DROP TABLE IF EXISTS books_categories;
DROP TABLE IF EXISTS books;
