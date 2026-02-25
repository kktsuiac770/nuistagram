package repository

import (
	"database/sql"
	"nuistagram/internal/cache"
	"nuistagram/internal/models"
	"time"
)

type UserRepository interface {
	GetByID(id int64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	GetAll() ([]models.User, error)
	Create(username, passwordHash string) (int64, error)
}

type NuiRepository interface {
	GetAll() ([]models.Nui, error)
	GetByUserID(userID int64) ([]models.Nui, error)
	GetOrCreate(name string, userID int64) (int64, error)
	InvalidateCache()
}

type PhotoRepository interface {
	GetAll(page int, currentUserID int64) (*PaginationResult, error)
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
	Users     UserRepository
	Nuis      NuiRepository
	Photos    PhotoRepository
	Albums    AlbumRepository
	Favorites FavoriteRepository
}

func NewRepositories(db *sql.DB, cache *cache.Cache) *Repositories {
	return &Repositories{
		Users:     NewUserRepository(db),
		Nuis:      NewCachedNuiRepository(db, cache),
		Photos:    NewPhotoRepository(db),
		Albums:    NewAlbumRepository(db),
		Favorites: NewFavoriteRepository(db),
	}
}
