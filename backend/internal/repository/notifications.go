package repository

import (
	"database/sql"
	"nuistagram/internal/models"
)

type notificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(userID, actorID int64, notifType string, photoID, commentID int64) (int64, error) {
	if userID == actorID {
		return 0, nil
	}

	result, err := r.db.Exec(`
		INSERT INTO notifications (user_id, actor_id, type, photo_id, comment_id)
		VALUES (?, ?, ?, ?, ?)
	`, userID, actorID, notifType, photoID, commentID)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *notificationRepository) GetByUserID(userID int64, limit, offset int) ([]models.Notification, error) {
	rows, err := r.db.Query(`
		SELECT n.id, n.user_id, n.actor_id, n.type, n.photo_id, n.comment_id, n.read, n.created_at,
			u.id, u.username, COALESCE(u.avatar, '')
		FROM notifications n
		JOIN users u ON n.actor_id = u.id
		WHERE n.user_id = ?
		ORDER BY n.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var photoID, commentID sql.NullInt64
		var avatar sql.NullString
		var readInt int
		n.Actor = &models.User{}

		if err := rows.Scan(
			&n.ID, &n.UserID, &n.ActorID, &n.Type, &photoID, &commentID, &readInt, &n.CreatedAt,
			&n.Actor.ID, &n.Actor.Username, &avatar,
		); err != nil {
			return nil, err
		}
		if photoID.Valid {
			n.PhotoID = photoID.Int64
		}
		if commentID.Valid {
			n.CommentID = commentID.Int64
		}
		n.Actor.Avatar = avatar.String
		n.Read = readInt == 1
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (r *notificationRepository) GetUnreadCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read = 0",
		userID,
	).Scan(&count)
	return count, err
}

func (r *notificationRepository) MarkAsRead(notificationID, userID int64) error {
	_, err := r.db.Exec(
		"UPDATE notifications SET read = 1 WHERE id = ? AND user_id = ?",
		notificationID, userID,
	)
	return err
}

func (r *notificationRepository) MarkAllAsRead(userID int64) error {
	_, err := r.db.Exec(
		"UPDATE notifications SET read = 1 WHERE user_id = ?",
		userID,
	)
	return err
}
