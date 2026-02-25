package repository

import (
	"database/sql"
	"nuistagram/internal/cache"
	"nuistagram/internal/models"
	"time"
)

type UserRepository interface {
	GetByID(id int64) (*models.User, error)
	GetByIDWithCounts(id int64, currentUserID int64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetByUsernameWithCounts(username string, currentUserID int64) (*models.User, error)
	GetAll() ([]models.User, error)
	Search(query string, limit int) ([]models.User, error)
	Create(username, passwordHash string) (int64, error)
	UpdateProfile(userID int64, bio string) error
	UpdateAvatar(userID int64, avatar string) error
}

type NuiRepository interface {
	GetAll() ([]models.Nui, error)
	GetByUserID(userID int64) ([]models.Nui, error)
	GetOrCreate(name string, userID int64) (int64, error)
	InvalidateCache()
}

type PhotoRepository interface {
	GetAll(page int, currentUserID int64) (*PaginationResult, error)
	GetFollowingFeed(userID int64, page int) (*PaginationResult, error)
	GetByNui(nuiName string, page int, currentUserID int64) (*PaginationResult, error)
	GetByUser(userID int64, page int, currentUserID int64) (*PaginationResult, error)
	GetFavorites(userID int64, page int) (*PaginationResult, error)
	Search(tags []string, mode string, page int, currentUserID int64) (*PaginationResult, error)
	GetByID(photoID int64, currentUserID int64) (*models.PhotoWithNuis, string, error)
	GetForEdit(photoID int64, userID int64) (*models.PhotoWithNuis, error)
	Create(filename, thumbnail string, userID int64, description string, takenAt *time.Time) (int64, error)
	Update(photoID int64, userID int64, description string, takenAt *time.Time) error
	Delete(photoID int64) error
	GetFilenamesByUser(userID int64) ([]string, error)
}

type AlbumRepository interface {
	GetByUserID(userID int64) ([]models.Album, error)
	GetByID(albumID int64) (*models.AlbumWithPhotos, error)
	Create(name, description string, userID int64) (int64, error)
	Delete(albumID int64) error
	AddPhoto(albumID int64, photoID int64) error
}

type FavoriteRepository interface {
	IsFavorited(photoID, userID int64) bool
	Toggle(photoID, userID int64) (bool, error)
}

type FollowRepository interface {
	Follow(followerID, followingID int64) error
	Unfollow(followerID, followingID int64) error
	IsFollowing(followerID, followingID int64) bool
	GetFollowers(userID int64, limit int) ([]models.User, error)
	GetFollowing(userID int64, limit int) ([]models.User, error)
	GetFollowerCount(userID int64) (int, error)
	GetFollowingCount(userID int64) (int, error)
}

type LikeRepository interface {
	Toggle(photoID, userID int64) (bool, error)
	IsLiked(photoID, userID int64) bool
	GetLikeCount(photoID int64) (int, error)
	GetLikers(photoID int64, limit int) ([]models.User, error)
}

type CommentRepository interface {
	Create(photoID, userID int64, content string) (int64, error)
	GetByPhotoID(photoID int64, limit, offset int) ([]models.Comment, error)
	GetCount(photoID int64) (int, error)
	Delete(commentID, userID int64) error
	GetByID(commentID int64) (*models.Comment, error)
}

type NotificationRepository interface {
	Create(userID, actorID int64, notifType string, photoID, commentID int64) (int64, error)
	GetByUserID(userID int64, limit, offset int) ([]models.Notification, error)
	GetUnreadCount(userID int64) (int, error)
	MarkAsRead(notificationID, userID int64) error
	MarkAllAsRead(userID int64) error
}

type PaginationResult struct {
	Photos      []models.PhotoWithNuis `json:"photos"`
	CurrentPage int                    `json:"current_page"`
	TotalPages  int                    `json:"total_pages"`
	TotalCount  int                    `json:"total_count"`
	HasPrev     bool                   `json:"has_prev"`
	HasNext     bool                   `json:"has_next"`
	Pages       []int                  `json:"pages"`
}

type Repositories struct {
	Users         UserRepository
	Nuis          NuiRepository
	Photos        PhotoRepository
	Albums        AlbumRepository
	Favorites     FavoriteRepository
	Follows       FollowRepository
	Likes         LikeRepository
	Comments      CommentRepository
	Notifications NotificationRepository
}

func NewRepositories(db *sql.DB, cache *cache.Cache) *Repositories {
	return &Repositories{
		Users:         NewUserRepository(db),
		Nuis:          NewCachedNuiRepository(db, cache),
		Photos:        NewPhotoRepository(db),
		Albums:        NewAlbumRepository(db),
		Favorites:     NewFavoriteRepository(db),
		Follows:       NewFollowRepository(db),
		Likes:         NewLikeRepository(db),
		Comments:      NewCommentRepository(db),
		Notifications: NewNotificationRepository(db),
	}
}
