CREATE INDEX IF NOT EXISTS idx_photos_user_id ON photos (user_id);

CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments (user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_actor_id ON notifications (actor_id);
