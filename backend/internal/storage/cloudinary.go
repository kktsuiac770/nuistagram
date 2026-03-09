package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CloudinaryStorage persists images on Cloudinary using the signed upload API.
// The identifier stored in the database is the full HTTPS URL returned by
// Cloudinary, which the frontend uses directly as the image src.
type CloudinaryStorage struct {
	cloudName string
	apiKey    string
	apiSecret string
	// folder is a path prefix for all public IDs (e.g. "nuistagram").
	folder string
}

func NewCloudinaryStorage(cloudName, apiKey, apiSecret, folder string) *CloudinaryStorage {
	if folder == "" {
		folder = "nuistagram"
	}
	return &CloudinaryStorage{
		cloudName: cloudName,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		folder:    folder,
	}
}

// sign computes the SHA-1 signature required by the Cloudinary signed upload API.
// params must not include api_key, file, or resource_type.
func (s *CloudinaryStorage) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}

	toSign := strings.Join(parts, "&") + s.apiSecret
	sum := sha1.Sum([]byte(toSign))
	return fmt.Sprintf("%x", sum)
}

// publicIDFromFilename derives a Cloudinary public ID from a filename.
// E.g. "avatars/abc123.jpg" → "<folder>/avatars/abc123"
func (s *CloudinaryStorage) publicIDFromFilename(filename string) string {
	withoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))
	return s.folder + "/" + withoutExt
}

type cloudinaryUploadResponse struct {
	SecureURL string `json:"secure_url"`
	PublicID  string `json:"public_id"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *CloudinaryStorage) Upload(ctx context.Context, data []byte, filename string) (UploadResult, error) {
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	publicID := s.publicIDFromFilename(filename)

	params := map[string]string{
		"public_id": publicID,
		"timestamp": timestamp,
	}
	signature := s.sign(params)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("api_key", s.apiKey)
	mw.WriteField("timestamp", timestamp)
	mw.WriteField("public_id", publicID)
	mw.WriteField("signature", signature)

	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return UploadResult{}, err
	}
	if _, err = fw.Write(data); err != nil {
		return UploadResult{}, err
	}
	mw.Close()

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", s.cloudName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return UploadResult{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UploadResult{}, err
	}
	defer resp.Body.Close()

	var result cloudinaryUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return UploadResult{}, fmt.Errorf("cloudinary: decode response: %w", err)
	}
	if result.Error != nil {
		return UploadResult{}, fmt.Errorf("cloudinary: %s", result.Error.Message)
	}

	return UploadResult{URL: result.SecureURL}, nil
}

// Delete removes an asset from Cloudinary.
// identifier is the full Cloudinary HTTPS URL stored in the database.
func (s *CloudinaryStorage) Delete(ctx context.Context, identifier string) error {
	publicID := publicIDFromURL(identifier)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	params := map[string]string{
		"public_id": publicID,
		"timestamp": timestamp,
	}
	signature := s.sign(params)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("api_key", s.apiKey)
	mw.WriteField("timestamp", timestamp)
	mw.WriteField("public_id", publicID)
	mw.WriteField("signature", signature)
	mw.Close()

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", s.cloudName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// FetchContent downloads the asset from its Cloudinary URL.
func (s *CloudinaryStorage) FetchContent(ctx context.Context, identifier string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, identifier, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// publicIDFromURL extracts the Cloudinary public ID from a secure URL.
// e.g. "https://res.cloudinary.com/cloud/image/upload/v123/folder/name.jpg"
// → "folder/name"
func publicIDFromURL(rawURL string) string {
	const marker = "/upload/"
	idx := strings.Index(rawURL, marker)
	if idx == -1 {
		return rawURL
	}
	path := rawURL[idx+len(marker):]

	// Strip optional version segment (v<digits>/)
	if len(path) > 1 && path[0] == 'v' {
		slashIdx := strings.Index(path, "/")
		if slashIdx > 1 {
			allDigits := true
			for _, c := range path[1:slashIdx] {
				if c < '0' || c > '9' {
					allDigits = false
					break
				}
			}
			if allDigits {
				path = path[slashIdx+1:]
			}
		}
	}

	// Strip extension
	if dot := strings.LastIndex(path, "."); dot != -1 {
		path = path[:dot]
	}
	return path
}
