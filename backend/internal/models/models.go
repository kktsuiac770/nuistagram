package models

import "time"

type User struct {
	ID             int64
	Username       string
	PasswordHash   string
	Bio            string
	Avatar         string
	Role           string
	CreatedAt      time.Time
	PhotoCount     int
	FollowingCount int
	FollowerCount  int
	IsFollowing    bool
}

type Nui struct {
	ID        int64
	Name      string
	UserID    int64
	CreatedAt time.Time
}

type Photo struct {
	ID           int64
	Filename     string
	Thumbnail    string
	UserID       int64
	Description  string
	TakenAt      time.Time
	CreatedAt    time.Time
	IsFavorite   bool
	LikeCount    int
	IsLiked      bool
	CommentCount int
	Username     string
}

type PhotoWithNuis struct {
	Photo
	NuiNames []string
}

type Album struct {
	ID          int64
	Name        string
	Description string
	UserID      int64
	CreatedAt   time.Time
	PhotoCount  int
}

type AlbumWithPhotos struct {
	Album
	Photos []PhotoWithNuis
}

type Follow struct {
	FollowerID  int64
	FollowingID int64
	CreatedAt   time.Time
}

type Comment struct {
	ID        int64
	PhotoID   int64
	UserID    int64
	Content   string
	CreatedAt time.Time
	Username  string
}

type Notification struct {
	ID        int64
	UserID    int64
	ActorID   int64
	Type      string
	PhotoID   int64
	CommentID int64
	Read      bool
	CreatedAt time.Time
	Actor     *User
	Photo     *Photo
}
