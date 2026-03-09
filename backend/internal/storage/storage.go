package storage

import (
	"context"
	"fmt"
)

// UploadResult holds the result of a successful upload.
// URL is the identifier stored in the database.
// For local storage it is the bare relative filename (e.g. "abc123.jpg").
// For Cloudinary it is the full HTTPS URL.
type UploadResult struct {
	URL string
}

// Storage is the provider-agnostic interface for image persistence.
// All methods receive a context so callers can propagate cancellation.
//
// identifier in Delete / FetchContent is whatever was previously returned
// as UploadResult.URL by Upload — each implementation understands its own
// identifier format.
type Storage interface {
	// Upload persists data under the given filename and returns the identifier
	// to be stored in the database.
	Upload(ctx context.Context, data []byte, filename string) (UploadResult, error)

	// Delete removes the asset identified by the stored identifier.
	Delete(ctx context.Context, identifier string) error

	// FetchContent retrieves the raw bytes of the asset identified by the
	// stored identifier.
	FetchContent(ctx context.Context, identifier string) ([]byte, error)
}

// New creates a Storage implementation based on the supplied parameters.
// provider must be "local" or "cloudinary".
// uploadDir is used only for local storage (e.g. "static/uploads").
// cloudName, apiKey, apiSecret, folder are used only for Cloudinary.
func New(provider, uploadDir, cloudName, apiKey, apiSecret, folder string) (Storage, error) {
	switch provider {
	case "cloudinary":
		if cloudName == "" || apiKey == "" || apiSecret == "" {
			return nil, fmt.Errorf(
				"storage: cloudinary requires CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY, and CLOUDINARY_API_SECRET",
			)
		}
		return NewCloudinaryStorage(cloudName, apiKey, apiSecret, folder), nil
	default:
		return NewLocalStorage(uploadDir), nil
	}
}
