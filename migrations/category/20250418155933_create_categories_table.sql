-- migrate:up
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_categories_name ON categories(name);
CREATE INDEX IF NOT EXISTS idx_categories_parent_id ON categories(parent_id);
CREATE INDEX IF NOT EXISTS idx_categories_deleted_at ON categories(deleted_at);

-- Add basic category structure with parent categories
INSERT INTO categories (id, name, description) VALUES
  ('50c3ef9e-d1aa-4e88-aa75-7d92c9d11111', 'Fiction', 'Fiction books and novels'),
  ('50c3ef9e-d1aa-4e88-aa75-7d92c9d22222', 'Non-Fiction', 'Non-fiction and educational books'),
  ('50c3ef9e-d1aa-4e88-aa75-7d92c9d33333', 'Science', 'Science and research books')
ON CONFLICT (name) DO NOTHING;

-- Now add subcategories with references to parents
INSERT INTO categories (id, name, description, parent_id) VALUES
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d11111', 'Fantasy', 'Fantasy novels and stories', '50c3ef9e-d1aa-4e88-aa75-7d92c9d11111'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d22222', 'Science Fiction', 'Science fiction novels and stories', '50c3ef9e-d1aa-4e88-aa75-7d92c9d11111'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d33333', 'Mystery', 'Mystery and detective novels', '50c3ef9e-d1aa-4e88-aa75-7d92c9d11111'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d44444', 'Biography', 'Biographies and autobiographies', '50c3ef9e-d1aa-4e88-aa75-7d92c9d22222'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d55555', 'History', 'Historical books and records', '50c3ef9e-d1aa-4e88-aa75-7d92c9d22222'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d66666', 'Physics', 'Physics and related topics', '50c3ef9e-d1aa-4e88-aa75-7d92c9d33333'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d77777', 'Biology', 'Biology and life sciences', '50c3ef9e-d1aa-4e88-aa75-7d92c9d33333'),
  ('60c3ef9e-d1aa-4e88-aa75-7d92c9d88888', 'Computer Science', 'Computer science and programming', '50c3ef9e-d1aa-4e88-aa75-7d92c9d33333')
ON CONFLICT (name) DO NOTHING;

-- Insert sample book category relationships (in the category database as a reference)
CREATE TABLE IF NOT EXISTS books_categories_ref (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT books_categories_ref_book_id_category_id_key UNIQUE (book_id, category_id)
);

-- This is just a reference table for tracking which books use which categories
CREATE INDEX IF NOT EXISTS idx_books_categories_ref_book_id ON books_categories_ref(book_id);
CREATE INDEX IF NOT EXISTS idx_books_categories_ref_category_id ON books_categories_ref(category_id);

-- migrate:down
DROP TABLE IF EXISTS books_categories_ref;
DROP TABLE IF EXISTS categories;
