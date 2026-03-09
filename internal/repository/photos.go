package repository

import (
	"database/sql"
	"fmt"
	"nuistagram/internal/models"
	"strings"
	"time"
)

const PhotosPerPage = 20

func buildWordMatchConditions(tags []string) (string, []interface{}) {
	conditions := make([]string, len(tags))
	args := make([]interface{}, len(tags))
	for i, tag := range tags {
		conditions[i] = "(' ' || LOWER(n.name) || ' ') LIKE ('% ' || LOWER(?) || ' %')"
		args[i] = tag
	}
	return strings.Join(conditions, " OR "), args
}

func buildAndExistsConditions(tags []string) (string, []interface{}) {
	conditions := make([]string, len(tags))
	args := make([]interface{}, len(tags))
	for i, tag := range tags {
		conditions[i] = fmt.Sprintf(`EXISTS (
			SELECT 1 FROM photo_nuis pn JOIN nuis n ON pn.nui_id = n.id 
			WHERE pn.photo_id = p.id 
			AND (' ' || LOWER(n.name) || ' ') LIKE ('%% ' || LOWER(?) || ' %%')
		)`)
		args[i] = tag
	}
	return strings.Join(conditions, " AND "), args
}

type photoRepository struct {
	db *sql.DB
}

func NewPhotoRepository(db *sql.DB) PhotoRepository {
	return &photoRepository{db: db}
}

func (r *photoRepository) GetAll(page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery(nil, "", page, currentUserID, 0, false)
}

func (r *photoRepository) GetFollowingFeed(userID int64, page int) (*PaginationResult, error) {
	return r.getPhotosWithQuery(nil, "", page, userID, 0, true)
}

func (r *photoRepository) GetByNui(nuiName string, page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery([]string{nuiName}, "or", page, currentUserID, 0, false)
}

func (r *photoRepository) GetByUser(userID int64, page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery(nil, "", page, currentUserID, userID, false)
}

func (r *photoRepository) GetFavorites(userID int64, page int) (*PaginationResult, error) {
	return r.getFavoritePhotos(userID, page)
}

func (r *photoRepository) Search(tags []string, mode string, page int, currentUserID int64) (*PaginationResult, error) {
	if len(tags) == 0 {
		return r.GetAll(page, currentUserID)
	}
	return r.getPhotosWithQuery(tags, mode, page, currentUserID, 0, false)
}

func (r *photoRepository) countPhotos(tags []string, mode string, filterUserID int64, followingOnly bool, currentUserID int64) (int, error) {
	var query string
	var args []interface{}

	followingJoin := ""
	if followingOnly {
		followingJoin = "JOIN follows f ON p.user_id = f.following_id AND f.follower_id = ?"
		args = append(args, currentUserID)
	}

	if filterUserID > 0 {
		query = fmt.Sprintf(`SELECT COUNT(DISTINCT p.id) FROM photos p %s WHERE p.user_id = ?`, followingJoin)
		args = append(args, filterUserID)
	} else if len(tags) == 0 {
		query = fmt.Sprintf(`SELECT COUNT(DISTINCT p.id) FROM photos p %s`, followingJoin)
	} else if mode == "and" {
		conditions, condArgs := buildAndExistsConditions(tags)
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			%s
			WHERE %s
		`, followingJoin, conditions)
		args = append(args, condArgs...)

		var count int
		rows, err := r.db.Query(query, args...)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		for rows.Next() {
			count++
		}
		return count, nil
	} else {
		conditions, condArgs := buildWordMatchConditions(tags)
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			%s
			WHERE %s
		`, followingJoin, conditions)
		args = append(args, condArgs...)
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (r *photoRepository) getPhotosWithQuery(tags []string, mode string, page int, currentUserID int64, filterUserID int64, followingOnly bool) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}

	totalCount, err := r.countPhotos(tags, mode, filterUserID, followingOnly, currentUserID)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + PhotosPerPage - 1) / PhotosPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * PhotosPerPage

	var query string
	var args []interface{}

	followingJoin := ""
	if followingOnly {
		followingJoin = "JOIN follows f ON p.user_id = f.following_id AND f.follower_id = ?"
		args = append(args, currentUserID)
	}

	favoriteJoin := ""
	favoriteSelect := "0"
	if currentUserID > 0 {
		favoriteJoin = "LEFT JOIN favorites fav ON p.id = fav.photo_id AND fav.user_id = ?"
		favoriteSelect = "CASE WHEN fav.photo_id IS NOT NULL THEN 1 ELSE 0 END"
		args = append(args, currentUserID)
	}

	likeJoin := ""
	likeSelect := "0"
	if currentUserID > 0 {
		likeJoin = "LEFT JOIN likes l ON p.id = l.photo_id AND l.user_id = ?"
		likeSelect = "CASE WHEN l.photo_id IS NOT NULL THEN 1 ELSE 0 END"
		args = append(args, currentUserID)
	}

	baseSelect := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
			(SELECT GROUP_CONCAT(n.name, ',') FROM nuis n JOIN photo_nuis pn ON n.id = pn.nui_id WHERE pn.photo_id = p.id) as nui_names,
			%s as is_favorite,
			(SELECT COUNT(*) FROM likes WHERE photo_id = p.id) as like_count,
			%s as is_liked,
			(SELECT COUNT(*) FROM comments WHERE photo_id = p.id) as comment_count,
			u.username
		FROM photos p
		JOIN users u ON p.user_id = u.id
		%s
		%s
		%s
	`, favoriteSelect, likeSelect, followingJoin, favoriteJoin, likeJoin)

	if filterUserID > 0 {
		query = fmt.Sprintf(`%s WHERE p.user_id = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, baseSelect)
		args = append(args, filterUserID, PhotosPerPage, offset)
	} else if len(tags) == 0 {
		query = fmt.Sprintf(`%s ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, baseSelect)
		args = append(args, PhotosPerPage, offset)
	} else if mode == "and" {
		conditions, condArgs := buildAndExistsConditions(tags)
		query = fmt.Sprintf(`%s WHERE %s ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, baseSelect, conditions)
		args = append(args, condArgs...)
		args = append(args, PhotosPerPage, offset)
	} else {
		conditions, condArgs := buildWordMatchConditions(tags)
		query = fmt.Sprintf(`
			%s
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE %s
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, baseSelect, conditions)
		args = append(args, condArgs...)
		args = append(args, PhotosPerPage, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	photos := make([]models.PhotoWithNuis, 0)
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var description sql.NullString
		var nuiNamesStr sql.NullString
		var isFavorite, isLiked int

		err := rows.Scan(
			&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description,
			&takenAt, &p.CreatedAt, &nuiNamesStr, &isFavorite, &p.LikeCount, &isLiked, &p.CommentCount, &p.Username,
		)
		if err != nil {
			return nil, err
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if description.Valid {
			p.Description = description.String
		}
		if nuiNamesStr.Valid && nuiNamesStr.String != "" {
			p.NuiNames = strings.Split(nuiNamesStr.String, ",")
		} else {
			p.NuiNames = []string{}
		}
		p.IsFavorite = isFavorite == 1
		p.IsLiked = isLiked == 1
		photos = append(photos, p)
	}

	pages := calculatePaginationPages(page, totalPages)

	return &PaginationResult{
		Photos:      photos,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Pages:       pages,
	}, nil
}

func (r *photoRepository) getFavoritePhotos(userID int64, page int) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}

	var totalCount int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM favorites WHERE user_id = ?`,
		userID,
	).Scan(&totalCount)
	if err != nil {
		return nil, err
	}

	totalPages := (totalCount + PhotosPerPage - 1) / PhotosPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * PhotosPerPage

	query := `
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
			(SELECT GROUP_CONCAT(n.name, ',') FROM nuis n JOIN photo_nuis pn ON n.id = pn.nui_id WHERE pn.photo_id = p.id) as nui_names,
			(SELECT COUNT(*) FROM likes WHERE photo_id = p.id) as like_count,
			(SELECT COUNT(*) FROM comments WHERE photo_id = p.id) as comment_count,
			u.username
		FROM photos p
		JOIN favorites f ON p.id = f.photo_id
		JOIN users u ON p.user_id = u.id
		WHERE f.user_id = ?
		ORDER BY f.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, userID, PhotosPerPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	photos := make([]models.PhotoWithNuis, 0)
	for rows.Next() {
		var p models.PhotoWithNuis
		var takenAt sql.NullTime
		var description sql.NullString
		var nuiNamesStr sql.NullString
		err := rows.Scan(
			&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description,
			&takenAt, &p.CreatedAt, &nuiNamesStr, &p.LikeCount, &p.CommentCount, &p.Username,
		)
		if err != nil {
			return nil, err
		}
		if takenAt.Valid {
			p.TakenAt = takenAt.Time
		}
		if description.Valid {
			p.Description = description.String
		}
		if nuiNamesStr.Valid && nuiNamesStr.String != "" {
			p.NuiNames = strings.Split(nuiNamesStr.String, ",")
		} else {
			p.NuiNames = []string{}
		}
		p.IsFavorite = true
		photos = append(photos, p)
	}

	pages := calculatePaginationPages(page, totalPages)

	return &PaginationResult{
		Photos:      photos,
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		Pages:       pages,
	}, nil
}

func (r *photoRepository) GetByID(photoID int64, currentUserID int64) (*models.PhotoWithNuis, string, error) {
	var p models.PhotoWithNuis
	var takenAt sql.NullTime
	var description sql.NullString
	var username string

	err := r.db.QueryRow(`
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at, u.username,
			(SELECT COUNT(*) FROM likes WHERE photo_id = p.id) as like_count,
			(SELECT COUNT(*) FROM comments WHERE photo_id = p.id) as comment_count
		FROM photos p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`, photoID).Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description, &takenAt, &p.CreatedAt, &username, &p.LikeCount, &p.CommentCount)

	if err != nil {
		return nil, "", err
	}

	if takenAt.Valid {
		p.TakenAt = takenAt.Time
	}
	if description.Valid {
		p.Description = description.String
	}

	p.NuiNames, _ = r.getNuiNamesForPhoto(p.ID)
	if currentUserID > 0 {
		p.IsFavorite = r.isPhotoFavorited(p.ID, currentUserID)
		p.IsLiked = r.isPhotoLiked(p.ID, currentUserID)
	}
	p.Username = username

	return &p, username, nil
}

func (r *photoRepository) GetForEdit(photoID int64, userID int64) (*models.PhotoWithNuis, error) {
	var p models.PhotoWithNuis
	var takenAt sql.NullTime
	var description sql.NullString

	err := r.db.QueryRow(`
		SELECT id, filename, COALESCE(thumbnail, ''), user_id, description, taken_at, created_at
		FROM photos WHERE id = ? AND user_id = ?
	`, photoID, userID).Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description, &takenAt, &p.CreatedAt)

	if err != nil {
		return nil, err
	}

	if takenAt.Valid {
		p.TakenAt = takenAt.Time
	}
	if description.Valid {
		p.Description = description.String
	}

	p.NuiNames, _ = r.getNuiNamesForPhoto(p.ID)
	return &p, nil
}

func (r *photoRepository) Create(filename, thumbnail string, userID int64, description string, takenAt *time.Time) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO photos (filename, thumbnail, user_id, description, taken_at)
		VALUES (?, ?, ?, ?, ?)
	`, filename, thumbnail, userID, description, takenAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *photoRepository) Update(photoID int64, userID int64, description string, takenAt *time.Time) error {
	_, err := r.db.Exec(`
		UPDATE photos SET description = ?, taken_at = ? WHERE id = ? AND user_id = ?
	`, description, takenAt, photoID, userID)
	return err
}

func (r *photoRepository) Delete(photoID int64) error {
	_, err := r.db.Exec("DELETE FROM photos WHERE id = ?", photoID)
	return err
}

func (r *photoRepository) GetFilenamesByUser(userID int64) ([]string, error) {
	rows, err := r.db.Query(`SELECT filename FROM photos WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filenames := make([]string, 0)
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		filenames = append(filenames, f)
	}
	return filenames, nil
}

func (r *photoRepository) getNuiNamesForPhoto(photoID int64) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT n.name FROM nuis n
		JOIN photo_nuis pn ON n.id = pn.nui_id
		WHERE pn.photo_id = ?
		ORDER BY n.name
	`, photoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func (r *photoRepository) isPhotoFavorited(photoID, userID int64) bool {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM favorites WHERE photo_id = ? AND user_id = ?`,
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func (r *photoRepository) isPhotoLiked(photoID, userID int64) bool {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM likes WHERE photo_id = ? AND user_id = ?`,
		photoID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

func calculatePaginationPages(current, total int) []int {
	if total <= 0 {
		return []int{1}
	}
	if total <= 7 {
		pages := make([]int, total)
		for i := 0; i < total; i++ {
			pages[i] = i + 1
		}
		return pages
	}

	pages := make([]int, 0, 5)
	start := current - 2
	end := current + 2

	if start < 1 {
		end += 1 - start
		start = 1
	}
	if end > total {
		start -= end - total
		end = total
	}

	if start < 1 {
		start = 1
	}

	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	return pages
}

func (r *photoRepository) GetOwner(photoID int64) (string, string, int64, error) {
	var filename, thumbnail string
	var userID int64
	err := r.db.QueryRow(
		"SELECT filename, COALESCE(thumbnail, ''), user_id FROM photos WHERE id = ?",
		photoID,
	).Scan(&filename, &thumbnail, &userID)
	return filename, thumbnail, userID, err
}

func (r *photoRepository) SetNuis(photoID int64, nuiIDs []int64) error {
	_, err := r.db.Exec("DELETE FROM photo_nuis WHERE photo_id = ?", photoID)
	if err != nil {
		return err
	}
	for _, nuiID := range nuiIDs {
		_, err := r.db.Exec(
			"INSERT OR IGNORE INTO photo_nuis (photo_id, nui_id) VALUES (?, ?)",
			photoID, nuiID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
