CREATE TABLE
    IF NOT EXISTS users (
        id BIGSERIAL PRIMARY KEY,
        username TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        bio TEXT DEFAULT '',
        avatar TEXT DEFAULT '',
        created_at TIMESTAMPTZ DEFAULT NOW ()
    );

CREATE TABLE
    IF NOT EXISTS nuis (
        id BIGSERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        user_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        UNIQUE (name, user_id),
        FOREIGN KEY (user_id) REFERENCES users (id)
    );

CREATE TABLE
    IF NOT EXISTS photos (
        id BIGSERIAL PRIMARY KEY,
        filename TEXT NOT NULL,
        thumbnail TEXT,
        user_id BIGINT NOT NULL,
        description TEXT,
        taken_at DATE,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        FOREIGN KEY (user_id) REFERENCES users (id)
    );

CREATE TABLE
    IF NOT EXISTS photo_nuis (
        photo_id BIGINT NOT NULL,
        nui_id BIGINT NOT NULL,
        PRIMARY KEY (photo_id, nui_id),
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE,
        FOREIGN KEY (nui_id) REFERENCES nuis (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS favorites (
        user_id BIGINT NOT NULL,
        photo_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        PRIMARY KEY (user_id, photo_id),
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS likes (
        user_id BIGINT NOT NULL,
        photo_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        PRIMARY KEY (user_id, photo_id),
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS comments (
        id BIGSERIAL PRIMARY KEY,
        photo_id BIGINT NOT NULL,
        user_id BIGINT NOT NULL,
        content TEXT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE,
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS follows (
        follower_id BIGINT NOT NULL,
        following_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        PRIMARY KEY (follower_id, following_id),
        FOREIGN KEY (follower_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (following_id) REFERENCES users (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS notifications (
        id BIGSERIAL PRIMARY KEY,
        user_id BIGINT NOT NULL,
        actor_id BIGINT NOT NULL,
        type TEXT NOT NULL,
        photo_id BIGINT,
        comment_id BIGINT,
        read INTEGER DEFAULT 0,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE CASCADE,
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE,
        FOREIGN KEY (comment_id) REFERENCES comments (id) ON DELETE CASCADE
    );

CREATE TABLE
    IF NOT EXISTS albums (
        id BIGSERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        description TEXT,
        user_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        FOREIGN KEY (user_id) REFERENCES users (id)
    );

CREATE TABLE
    IF NOT EXISTS album_photos (
        album_id BIGINT NOT NULL,
        photo_id BIGINT NOT NULL,
        position INTEGER DEFAULT 0,
        PRIMARY KEY (album_id, photo_id),
        FOREIGN KEY (album_id) REFERENCES albums (id) ON DELETE CASCADE,
        FOREIGN KEY (photo_id) REFERENCES photos (id) ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS idx_likes_photo_id ON likes (photo_id);

CREATE INDEX IF NOT EXISTS idx_comments_photo_id ON comments (photo_id);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);

CREATE INDEX IF NOT EXISTS idx_follows_follower_id ON follows (follower_id);

CREATE INDEX IF NOT EXISTS idx_follows_following_id ON follows (following_id)