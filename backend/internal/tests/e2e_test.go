package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func setupE2ETest(t *testing.T) *TestServer {
	mux := http.NewServeMux()

	ts, err := SetupTestServer(mux)
	assert.NoError(t, err)

	srv := ts.Srv

	mux.HandleFunc("/api/photos", srv.APIGetPhotos)
	mux.HandleFunc("/api/photo/{id}", srv.APIGetPhoto)
	mux.HandleFunc("/api/me", srv.APIGetMe)
	mux.HandleFunc("/api/nuis", srv.APIGetNuis)
	mux.HandleFunc("/api/photo/{id}/like", srv.APIToggleLike)
	mux.HandleFunc("/api/user/{username}", srv.APIGetUser)
	mux.HandleFunc("/api/user/{username}/photos", srv.APIGetUserPhotos)
	mux.HandleFunc("POST /login", srv.Login)
	mux.HandleFunc("POST /register", srv.Register)
	mux.HandleFunc("/logout", srv.Logout)

	return ts
}

func TestE2E_RegisterAndLogin(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	registerData := url.Values{}
	registerData.Set("username", "testuser")
	registerData.Set("password", "Password123")

	resp, err := ts.Client.PostForm(ts.URL()+"/register", registerData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var registerResp map[string]bool
	json.NewDecoder(resp.Body).Decode(&registerResp)
	assert.True(t, registerResp["success"])
	resp.Body.Close()

	loginData := url.Values{}
	loginData.Set("username", "testuser")
	loginData.Set("password", "Password123")

	resp, err = ts.Client.PostForm(ts.URL()+"/login", loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResp map[string]bool
	json.NewDecoder(resp.Body).Decode(&loginResp)
	assert.True(t, loginResp["success"])

	cookies := resp.Cookies()
	assert.NotEmpty(t, cookies)
	assert.Equal(t, "session", cookies[0].Name)
	resp.Body.Close()
}

func TestE2E_RegisterDuplicateUsername(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	registerData := url.Values{}
	registerData.Set("username", "duplicateuser")
	registerData.Set("password", "Password123")

	resp, err := ts.Client.PostForm(ts.URL()+"/register", registerData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = ts.Client.PostForm(ts.URL()+"/register", registerData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_LoginInvalidCredentials(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	loginData := url.Values{}
	loginData.Set("username", "nonexistent")
	loginData.Set("password", "Password123")

	resp, err := ts.Client.PostForm(ts.URL()+"/login", loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_GetPhotosUnauthenticated(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/photos")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result, "photos")
	resp.Body.Close()
}

func TestE2E_GetMeUnauthenticated(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/me")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_GetNuis(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/nuis")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var nuis []interface{}
	json.NewDecoder(resp.Body).Decode(&nuis)
	resp.Body.Close()
}

func TestE2E_PhotoNotFound(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/photo/99999")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_UserNotFound(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/user/nonexistentuser")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestE2E_FullUserFlow(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	registerData := url.Values{}
	registerData.Set("username", "flowuser")
	registerData.Set("password", "Password123")

	resp, err := ts.Client.PostForm(ts.URL()+"/register", registerData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	loginData := url.Values{}
	loginData.Set("username", "flowuser")
	loginData.Set("password", "Password123")

	resp, err = ts.Client.PostForm(ts.URL()+"/login", loginData)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	sessionCookie := resp.Cookies()[0]

	req, _ := http.NewRequest("GET", ts.URL()+"/api/me", nil)
	req.AddCookie(sessionCookie)
	resp, err = ts.Client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var meResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&meResp)
	assert.Equal(t, "flowuser", meResp["username"])
	resp.Body.Close()
}

func TestE2E_GetPhotosWithPagination(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	resp, err := ts.Client.Get(ts.URL() + "/api/photos?page=1")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Contains(t, result, "photos")
	assert.Contains(t, result, "current_page")
	resp.Body.Close()
}

func TestE2E_PhotoWithTestUser(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password123"), bcrypt.DefaultCost)
	userID, err := CreateTestUser(ts.DB, "photouser", string(hashedPassword))
	assert.NoError(t, err)

	photoID, err := CreateTestPhoto(ts.DB, userID, "test.jpg", "Test photo")
	assert.NoError(t, err)

	nuiID, err := CreateTestNui(ts.DB, "testnui", userID)
	assert.NoError(t, err)

	err = LinkPhotoNui(ts.DB, photoID, nuiID)
	assert.NoError(t, err)

	resp, err := ts.Client.Get(ts.URL() + "/api/photos")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	photos := result["photos"].([]interface{})
	assert.GreaterOrEqual(t, len(photos), 1)
	resp.Body.Close()
}

func TestE2E_LikeRequiresAuth(t *testing.T) {
	ts := setupE2ETest(t)
	defer ts.Cleanup()

	req, _ := http.NewRequest("POST", ts.URL()+"/api/photo/1/like", bytes.NewReader(nil))
	resp, err := ts.Client.Do(req)
	assert.NoError(t, err)
	// Without CSRF middleware in e2e test setup, will get 401 from handler
	assert.True(t, resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden)
	resp.Body.Close()
}
