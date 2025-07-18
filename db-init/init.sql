CREATE TABLE IF NOT EXISTS people (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    age INTEGER,
    gender VARCHAR(10),
    nationality VARCHAR(3),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS emails (
    id SERIAL PRIMARY KEY,
    person_id INTEGER REFERENCES people(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS friendships (
    id SERIAL PRIMARY KEY,
    person_id INTEGER REFERENCES people(id) ON DELETE CASCADE,
    friend_id INTEGER REFERENCES people(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(person_id, friend_id)
    );

CREATE INDEX IF NOT EXISTS idx_people_last_name ON people(last_name);
CREATE INDEX IF NOT EXISTS idx_people_full_name ON people(first_name, last_name);
CREATE INDEX IF NOT EXISTS idx_emails_person_id ON emails(person_id);
CREATE INDEX IF NOT EXISTS idx_friendships_person_id ON friendships(person_id);
CREATE INDEX IF NOT EXISTS idx_friendships_friend_id ON friendships(friend_id);

-- Trigger для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_people_updated_at BEFORE UPDATE ON people
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();