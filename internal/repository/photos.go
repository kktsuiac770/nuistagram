package repository

import (
	"database/sql"
	"fmt"
	"nuistagram/internal/models"
	"strings"
	"time"
)

const PhotosPerPage = 20

type photoRepository struct {
	db *sql.DB
}

func NewPhotoRepository(db *sql.DB) PhotoRepository {
	return &photoRepository{db: db}
}

func (r *photoRepository) GetAll(page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery(nil, "", page, currentUserID)
}

func (r *photoRepository) GetByNui(nuiName string, page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery([]string{nuiName}, "or", page, currentUserID)
}

func (r *photoRepository) GetByUser(userID int64, page int, currentUserID int64) (*PaginationResult, error) {
	return r.getPhotosWithQuery(nil, "", page, currentUserID, userID)
}

func (r *photoRepository) GetFavorites(userID int64, page int) (*PaginationResult, error) {
	return r.getFavoritePhotos(userID, page)
}

func (r *photoRepository) Search(tags []string, mode string, page int, currentUserID int64) (*PaginationResult, error) {
	if len(tags) == 0 {
		return r.GetAll(page, currentUserID)
	}
	return r.getPhotosWithQuery(tags, mode, page, currentUserID)
}

func (r *photoRepository) countPhotos(tags []string, mode string, filterUserID int64) (int, error) {
	var query string
	var args []interface{}

	if filterUserID > 0 {
		query = `SELECT COUNT(DISTINCT p.id) FROM photos p WHERE p.user_id = ?`
		args = append(args, filterUserID)
	} else if len(tags) == 0 {
		query = `SELECT COUNT(DISTINCT p.id) FROM photos p`
	} else if mode == "and" {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
			GROUP BY p.id
			HAVING COUNT(DISTINCT n.id) = ?
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, len(tags))

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
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT COUNT(DISTINCT p.id)
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			WHERE n.name IN (%s)
		`, placeholders)
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (r *photoRepository) getPhotosWithQuery(tags []string, mode string, page int, currentUserID int64, filterUserID ...int64) (*PaginationResult, error) {
	if page < 1 {
		page = 1
	}

	var fUserID int64
	if len(filterUserID) > 0 {
		fUserID = filterUserID[0]
	}

	totalCount, err := r.countPhotos(tags, mode, fUserID)
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

	favoriteJoin := ""
	favoriteSelect := "0"
	if currentUserID > 0 {
		favoriteJoin = "LEFT JOIN favorites f ON p.id = f.photo_id AND f.user_id = ?"
		favoriteSelect = "CASE WHEN f.photo_id IS NOT NULL THEN 1 ELSE 0 END"
	}

	if fUserID > 0 {
		query = fmt.Sprintf(`
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
				(SELECT GROUP_CONCAT(n.name, ',') FROM nuis n JOIN photo_nuis pn ON n.id = pn.nui_id WHERE pn.photo_id = p.id) as nui_names,
				%s as is_favorite
			FROM photos p
			%s
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, favoriteSelect, favoriteJoin)
		if currentUserID > 0 {
			args = []interface{}{currentUserID, fUserID, PhotosPerPage, offset}
		} else {
			args = []interface{}{fUserID, PhotosPerPage, offset}
		}
	} else if len(tags) == 0 {
		query = fmt.Sprintf(`
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
				(SELECT GROUP_CONCAT(n.name, ',') FROM nuis n JOIN photo_nuis pn ON n.id = pn.nui_id WHERE pn.photo_id = p.id) as nui_names,
				%s as is_favorite
			FROM photos p
			%s
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, favoriteSelect, favoriteJoin)
		if currentUserID > 0 {
			args = []interface{}{currentUserID, PhotosPerPage, offset}
		} else {
			args = []interface{}{PhotosPerPage, offset}
		}
	} else if mode == "and" {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
				(SELECT GROUP_CONCAT(n2.name, ',') FROM nuis n2 JOIN photo_nuis pn2 ON n2.id = pn2.nui_id WHERE pn2.photo_id = p.id) as nui_names,
				%s as is_favorite
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			%s
			WHERE n.name IN (%s)
			GROUP BY p.id
			HAVING COUNT(DISTINCT n.id) = ?
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, favoriteSelect, favoriteJoin, placeholders)
		if currentUserID > 0 {
			args = []interface{}{currentUserID}
		} else {
			args = nil
		}
		for _, tag := range tags {
			args = append(args, tag)
		}
		args = append(args, len(tags), PhotosPerPage, offset)
	} else {
		placeholders := strings.Repeat("?,", len(tags))
		placeholders = placeholders[:len(placeholders)-1]
		query = fmt.Sprintf(`
			SELECT DISTINCT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at,
				(SELECT GROUP_CONCAT(n2.name, ',') FROM nuis n2 JOIN photo_nuis pn2 ON n2.id = pn2.nui_id WHERE pn2.photo_id = p.id) as nui_names,
				%s as is_favorite
			FROM photos p
			JOIN photo_nuis pn ON p.id = pn.photo_id
			JOIN nuis n ON pn.nui_id = n.id
			%s
			WHERE n.name IN (%s)
			ORDER BY p.created_at DESC
			LIMIT ? OFFSET ?
		`, favoriteSelect, favoriteJoin, placeholders)
		if currentUserID > 0 {
			args = []interface{}{currentUserID}
		} else {
			args = nil
		}
		for _, tag := range tags {
			args = append(args, tag)
		}
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
		var isFavorite int

		err := rows.Scan(
			&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description,
			&takenAt, &p.CreatedAt, &nuiNamesStr, &isFavorite,
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
			(SELECT GROUP_CONCAT(n.name, ',') FROM nuis n JOIN photo_nuis pn ON n.id = pn.nui_id WHERE pn.photo_id = p.id) as nui_names
		FROM photos p
		JOIN favorites f ON p.id = f.photo_id
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
			&takenAt, &p.CreatedAt, &nuiNamesStr,
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
		SELECT p.id, p.filename, COALESCE(p.thumbnail, ''), p.user_id, p.description, p.taken_at, p.created_at, u.username
		FROM photos p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?
	`, photoID).Scan(&p.ID, &p.Filename, &p.Thumbnail, &p.UserID, &description, &takenAt, &p.CreatedAt, &username)

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
	}

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
