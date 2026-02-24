package models

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	PhotoCount   int
}

type Nui struct {
	ID        int64
	Name      string
	UserID    int64
	CreatedAt time.Time
}

type Photo struct {
	ID          int64
	Filename    string
	Thumbnail   string
	UserID      int64
	Description string
	TakenAt     time.Time
	CreatedAt   time.Time
	IsFavorite  bool
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
