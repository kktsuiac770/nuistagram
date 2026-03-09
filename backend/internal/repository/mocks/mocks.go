package mocks

import (
	"database/sql"
	"nuistagram/internal/models"
	"nuistagram/internal/repository"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(id int64) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByIDWithCounts(id int64, currentUserID int64) (*models.User, error) {
	args := m.Called(id, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsernameWithCounts(username string, currentUserID int64) (*models.User, error) {
	args := m.Called(username, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll() ([]models.User, error) {
	args := m.Called()
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) Search(query string, limit int) ([]models.User, error) {
	args := m.Called(query, limit)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) Create(username, passwordHash string) (int64, error) {
	args := m.Called(username, passwordHash)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) UpdateProfile(userID int64, bio string) error {
	args := m.Called(userID, bio)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateAvatar(userID int64, avatar string) error {
	args := m.Called(userID, avatar)
	return args.Error(0)
}

type MockPhotoRepository struct {
	mock.Mock
}

func (m *MockPhotoRepository) GetAll(page int, currentUserID int64) (*repository.PaginationResult, error) {
	args := m.Called(page, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) GetFollowingFeed(userID int64, page int) (*repository.PaginationResult, error) {
	args := m.Called(userID, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) GetByNui(nuiName string, page int, currentUserID int64) (*repository.PaginationResult, error) {
	args := m.Called(nuiName, page, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) GetByUser(userID int64, page int, currentUserID int64) (*repository.PaginationResult, error) {
	args := m.Called(userID, page, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) GetFavorites(userID int64, page int) (*repository.PaginationResult, error) {
	args := m.Called(userID, page)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) Search(tags []string, mode string, page int, currentUserID int64) (*repository.PaginationResult, error) {
	args := m.Called(tags, mode, page, currentUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PaginationResult), args.Error(1)
}

func (m *MockPhotoRepository) GetByID(photoID int64, currentUserID int64) (*models.PhotoWithNuis, string, error) {
	args := m.Called(photoID, currentUserID)
	if args.Get(0) == nil {
		return nil, "", args.Error(2)
	}
	return args.Get(0).(*models.PhotoWithNuis), args.String(1), args.Error(2)
}

func (m *MockPhotoRepository) GetForEdit(photoID int64, userID int64) (*models.PhotoWithNuis, error) {
	args := m.Called(photoID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PhotoWithNuis), args.Error(1)
}

func (m *MockPhotoRepository) Create(filename, thumbnail string, userID int64, description string, takenAt *time.Time) (int64, error) {
	args := m.Called(filename, thumbnail, userID, description, takenAt)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPhotoRepository) Update(photoID int64, userID int64, description string, takenAt *time.Time) error {
	args := m.Called(photoID, userID, description, takenAt)
	return args.Error(0)
}

func (m *MockPhotoRepository) Delete(photoID int64) error {
	args := m.Called(photoID)
	return args.Error(0)
}

func (m *MockPhotoRepository) GetFilenamesByUser(userID int64) ([]string, error) {
	args := m.Called(userID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPhotoRepository) GetOwner(photoID int64) (string, string, int64, error) {
	args := m.Called(photoID)
	return args.String(0), args.String(1), args.Get(2).(int64), args.Error(3)
}

func (m *MockPhotoRepository) SetNuis(photoID int64, nuiIDs []int64) error {
	args := m.Called(photoID, nuiIDs)
	return args.Error(0)
}

type MockLikeRepository struct {
	mock.Mock
}

func (m *MockLikeRepository) Toggle(photoID, userID int64) (bool, error) {
	args := m.Called(photoID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockLikeRepository) IsLiked(photoID, userID int64) bool {
	args := m.Called(photoID, userID)
	return args.Bool(0)
}

func (m *MockLikeRepository) GetLikeCount(photoID int64) (int, error) {
	args := m.Called(photoID)
	return args.Int(0), args.Error(1)
}

func (m *MockLikeRepository) GetLikers(photoID int64, limit int) ([]models.User, error) {
	args := m.Called(photoID, limit)
	return args.Get(0).([]models.User), args.Error(1)
}

type MockFollowRepository struct {
	mock.Mock
}

func (m *MockFollowRepository) Follow(followerID, followingID int64) error {
	args := m.Called(followerID, followingID)
	return args.Error(0)
}

func (m *MockFollowRepository) Unfollow(followerID, followingID int64) error {
	args := m.Called(followerID, followingID)
	return args.Error(0)
}

func (m *MockFollowRepository) IsFollowing(followerID, followingID int64) bool {
	args := m.Called(followerID, followingID)
	return args.Bool(0)
}

func (m *MockFollowRepository) GetFollowers(userID int64, limit int) ([]models.User, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockFollowRepository) GetFollowing(userID int64, limit int) ([]models.User, error) {
	args := m.Called(userID, limit)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockFollowRepository) GetFollowerCount(userID int64) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func (m *MockFollowRepository) GetFollowingCount(userID int64) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

type MockCommentRepository struct {
	mock.Mock
}

func (m *MockCommentRepository) Create(photoID, userID int64, content string) (int64, error) {
	args := m.Called(photoID, userID, content)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCommentRepository) GetByPhotoID(photoID int64, limit, offset int) ([]models.Comment, error) {
	args := m.Called(photoID, limit, offset)
	return args.Get(0).([]models.Comment), args.Error(1)
}

func (m *MockCommentRepository) GetCount(photoID int64) (int, error) {
	args := m.Called(photoID)
	return args.Int(0), args.Error(1)
}

func (m *MockCommentRepository) Delete(commentID, userID int64) error {
	args := m.Called(commentID, userID)
	return args.Error(0)
}

func (m *MockCommentRepository) GetByID(commentID int64) (*models.Comment, error) {
	args := m.Called(commentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Comment), args.Error(1)
}

type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) Create(userID, actorID int64, notifType string, photoID, commentID int64) (int64, error) {
	args := m.Called(userID, actorID, notifType, photoID, commentID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationRepository) GetByUserID(userID int64, limit, offset int) ([]models.Notification, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]models.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetUnreadCount(userID int64) (int, error) {
	args := m.Called(userID)
	return args.Int(0), args.Error(1)
}

func (m *MockNotificationRepository) MarkAsRead(notificationID, userID int64) error {
	args := m.Called(notificationID, userID)
	return args.Error(0)
}

func (m *MockNotificationRepository) MarkAllAsRead(userID int64) error {
	args := m.Called(userID)
	return args.Error(0)
}

type MockNuiRepository struct {
	mock.Mock
}

func (m *MockNuiRepository) GetAll() ([]models.Nui, error) {
	args := m.Called()
	return args.Get(0).([]models.Nui), args.Error(1)
}

func (m *MockNuiRepository) GetByUserID(userID int64) ([]models.Nui, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Nui), args.Error(1)
}

func (m *MockNuiRepository) GetOrCreate(name string, userID int64) (int64, error) {
	args := m.Called(name, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNuiRepository) InvalidateCache() {
	m.Called()
}

type MockFavoriteRepository struct {
	mock.Mock
}

func (m *MockFavoriteRepository) IsFavorited(photoID, userID int64) bool {
	args := m.Called(photoID, userID)
	return args.Bool(0)
}

func (m *MockFavoriteRepository) Toggle(photoID, userID int64) (bool, error) {
	args := m.Called(photoID, userID)
	return args.Bool(0), args.Error(1)
}

type MockAlbumRepository struct {
	mock.Mock
}

func (m *MockAlbumRepository) GetByUserID(userID int64) ([]models.Album, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Album), args.Error(1)
}

func (m *MockAlbumRepository) GetByID(albumID int64) (*models.AlbumWithPhotos, error) {
	args := m.Called(albumID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AlbumWithPhotos), args.Error(1)
}

func (m *MockAlbumRepository) Create(name, description string, userID int64) (int64, error) {
	args := m.Called(name, description, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAlbumRepository) Delete(albumID int64) error {
	args := m.Called(albumID)
	return args.Error(0)
}

func (m *MockAlbumRepository) AddPhoto(albumID int64, photoID int64) error {
	args := m.Called(albumID, photoID)
	return args.Error(0)
}

func (m *MockAlbumRepository) GetOwnerID(albumID int64) (int64, error) {
	args := m.Called(albumID)
	return args.Get(0).(int64), args.Error(1)
}

func NewMockRepositories() *MockRepositories {
	return &MockRepositories{
		Users:         new(MockUserRepository),
		Photos:        new(MockPhotoRepository),
		Likes:         new(MockLikeRepository),
		Follows:       new(MockFollowRepository),
		Comments:      new(MockCommentRepository),
		Notifications: new(MockNotificationRepository),
		Nuis:          new(MockNuiRepository),
		Favorites:     new(MockFavoriteRepository),
		Albums:        new(MockAlbumRepository),
	}
}

type MockRepositories struct {
	Users         *MockUserRepository
	Photos        *MockPhotoRepository
	Likes         *MockLikeRepository
	Follows       *MockFollowRepository
	Comments      *MockCommentRepository
	Notifications *MockNotificationRepository
	Nuis          *MockNuiRepository
	Favorites     *MockFavoriteRepository
	Albums        *MockAlbumRepository
}

var _ error = sql.ErrNoRows
