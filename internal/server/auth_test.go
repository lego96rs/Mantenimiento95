package server

import (
	"context"
	"mantenimiento/internal/db"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"mantenimiento/internal/auth"
	"mantenimiento/internal/models"
)

type testAuthServer struct {
	h  http.Handler
	db *db.DB
}

func newAuthTestServer(t *testing.T) *testAuthServer {
	t.Helper()
	handler, database := newTestServerAndDB(t)
	return &testAuthServer{
		h:  handler,
		db: database,
	}
}

func (ts *testAuthServer) createUser(t *testing.T, username, password, role string, mustChange bool) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := models.CreateUser(context.Background(), ts.db, username, "Nombre "+username, role, hash, mustChange); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

func (ts *testAuthServer) do(t *testing.T, method, path, cookie string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	ts.h.ServeHTTP(rec, req)
	return rec
}

func (ts *testAuthServer) login(t *testing.T, username, password string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	rec := ts.do(t, http.MethodPost, "/login", "", url.Values{
		"username": {username},
		"password": {password},
	})
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.CookieName && cookie.Value != "" {
			return rec, cookie.Value
		}
	}
	return rec, ""
}

var csrfRe = regexp.MustCompile(`name="csrf" value="([^"]+)"`)

func extractCSRF(t *testing.T, body string) string {
	t.Helper()
	match := csrfRe.FindStringSubmatch(body)
	if match == nil {
		t.Fatal("no csrf field found in page")
	}
	return match[1]
}

func TestLoginPageRenders(t *testing.T) {
	ts := newAuthTestServer(t)
	rec := ts.do(t, http.MethodGet, "/login", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Usuario") {
		t.Fatalf("login page missing username field")
	}
}

func TestLoginSuccessSetsSessionAndHomeGreets(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)

	rec, cookie := ts.login(t, "ana", "clave-segura-1")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("login: status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
	if cookie == "" {
		t.Fatal("no session cookie set on successful login")
	}

	home := ts.do(t, http.MethodGet, "/", cookie, nil)
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "Nombre ana") {
		t.Fatalf("home after login: status=%d body=%s", home.Code, home.Body.String())
	}
}

func TestLoginFailuresAreGenericAndIdentical(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)

	wrongPass, c1 := ts.login(t, "ana", "clave-mala")
	noUser, c2 := ts.login(t, "nadie", "clave-mala")
	if c1 != "" || c2 != "" {
		t.Fatal("failed login produced a session cookie")
	}
	if wrongPass.Code != http.StatusUnauthorized || noUser.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected statuses: %d / %d", wrongPass.Code, noUser.Code)
	}

	normalize := func(body, user string) string { return strings.ReplaceAll(body, user, "USER") }
	if normalize(wrongPass.Body.String(), "ana") != normalize(noUser.Body.String(), "nadie") {
		t.Fatal("wrong-password and unknown-user responses differ")
	}
}

func TestInactiveUserCannotLogin(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	if _, err := ts.db.Write.Exec(`UPDATE users SET active = 0 WHERE username = 'ana'`); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	rec, cookie := ts.login(t, "ana", "clave-segura-1")
	if cookie != "" || rec.Code != http.StatusUnauthorized {
		t.Fatalf("inactive user logged in: status=%d cookie=%q", rec.Code, cookie)
	}
}

func TestLoginLockoutAfterRepeatedFailures(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)

	for i := 0; i < 5; i++ {
		rec, _ := ts.login(t, "ana", "clave-mala")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}

	rec, cookie := ts.login(t, "ana", "clave-segura-1")
	if rec.Code != http.StatusTooManyRequests || cookie != "" {
		t.Fatalf("after lockout: status=%d cookie=%q", rec.Code, cookie)
	}
}

func TestAnonymousIsRedirectedToLogin(t *testing.T) {
	ts := newAuthTestServer(t)
	for _, path := range []string{"/", "/password", "/admin"} {
		rec := ts.do(t, http.MethodGet, path, "", nil)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Fatalf("GET %s anonymous: status=%d location=%q", path, rec.Code, rec.Header().Get("Location"))
		}
	}
}

func TestTechnicianCannotAccessAdmin(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	ts.createUser(t, "jefa", "clave-segura-2", models.RoleAdmin, false)

	_, techCookie := ts.login(t, "ana", "clave-segura-1")
	_, adminCookie := ts.login(t, "jefa", "clave-segura-2")

	if rec := ts.do(t, http.MethodGet, "/admin", techCookie, nil); rec.Code != http.StatusForbidden {
		t.Fatalf("technician on /admin: status=%d, want 403", rec.Code)
	}
	if rec := ts.do(t, http.MethodGet, "/admin", adminCookie, nil); rec.Code != http.StatusOK {
		t.Fatalf("admin on /admin: status=%d, want 200", rec.Code)
	}
}

func TestPostWithoutCSRFTokenIsRejected(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	_, cookie := ts.login(t, "ana", "clave-segura-1")

	rec := ts.do(t, http.MethodPost, "/password", cookie, url.Values{
		"current": {"clave-segura-1"},
		"new":     {"otra-clave-99"},
		"confirm": {"otra-clave-99"},
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("POST without csrf: status=%d, want 403", rec.Code)
	}
}

func TestCrossSitePostIsRejected(t *testing.T) {
	ts := newAuthTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("username=ana&password=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()
	ts.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site POST: status=%d, want 403", rec.Code)
	}
}

func TestPasswordChangeFlow(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	_, cookie := ts.login(t, "ana", "clave-segura-1")

	page := ts.do(t, http.MethodGet, "/password", cookie, nil)
	csrf := extractCSRF(t, page.Body.String())

	rec := ts.do(t, http.MethodPost, "/password", cookie, url.Values{
		"csrf":    {csrf},
		"current": {"clave-segura-1"},
		"new":     {"nueva-clave-99"},
		"confirm": {"nueva-clave-99"},
	})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("password change: status=%d location=%q body=%s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}

	if home := ts.do(t, http.MethodGet, "/", cookie, nil); home.Code != http.StatusSeeOther {
		t.Fatalf("old session still valid after password change: status=%d", home.Code)
	}
	if rec, c := ts.login(t, "ana", "clave-segura-1"); c != "" {
		t.Fatalf("old password still works: status=%d", rec.Code)
	}
	if _, c := ts.login(t, "ana", "nueva-clave-99"); c == "" {
		t.Fatal("new password does not work")
	}
}

func TestPasswordChangeValidations(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	_, cookie := ts.login(t, "ana", "clave-segura-1")
	page := ts.do(t, http.MethodGet, "/password", cookie, nil)
	csrf := extractCSRF(t, page.Body.String())

	cases := []struct {
		name                  string
		current, new, confirm string
	}{
		{"wrong current", "clave-mala", "nueva-clave-99", "nueva-clave-99"},
		{"too short", "clave-segura-1", "corta", "corta"},
		{"mismatch", "clave-segura-1", "nueva-clave-99", "nueva-clave-98"},
		{"equals username", "clave-segura-1", "ana", "ana"},
		{"same as current", "clave-segura-1", "clave-segura-1", "clave-segura-1"},
	}

	for _, tc := range cases {
		rec := ts.do(t, http.MethodPost, "/password", cookie, url.Values{
			"csrf":    {csrf},
			"current": {tc.current},
			"new":     {tc.new},
			"confirm": {tc.confirm},
		})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s: status=%d, want 422", tc.name, rec.Code)
		}
	}
}

func TestMustChangePasswordForcesRedirect(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "nuevo", "clave-temporal-1", models.RoleTechnician, true)

	rec, cookie := ts.login(t, "nuevo", "clave-temporal-1")
	if rec.Header().Get("Location") != "/password" {
		t.Fatalf("login with temp password: location=%q, want /password", rec.Header().Get("Location"))
	}

	if home := ts.do(t, http.MethodGet, "/", cookie, nil); home.Header().Get("Location") != "/password" {
		t.Fatalf("GET / with must_change: location=%q, want /password", home.Header().Get("Location"))
	}

	page := ts.do(t, http.MethodGet, "/password", cookie, nil)
	csrf := extractCSRF(t, page.Body.String())
	change := ts.do(t, http.MethodPost, "/password", cookie, url.Values{
		"csrf":    {csrf},
		"current": {"clave-temporal-1"},
		"new":     {"definitiva-123"},
		"confirm": {"definitiva-123"},
	})
	if change.Code != http.StatusSeeOther {
		t.Fatalf("change temp password: status=%d body=%s", change.Code, change.Body.String())
	}

	rec2, cookie2 := ts.login(t, "nuevo", "definitiva-123")
	if rec2.Header().Get("Location") != "/" || cookie2 == "" {
		t.Fatalf("login after change: location=%q cookie=%q", rec2.Header().Get("Location"), cookie2)
	}
}

func TestLogout(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	_, cookie := ts.login(t, "ana", "clave-segura-1")

	home := ts.do(t, http.MethodGet, "/", cookie, nil)
	csrf := extractCSRF(t, home.Body.String())

	rec := ts.do(t, http.MethodPost, "/logout", cookie, url.Values{"csrf": {csrf}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("logout: status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
	if after := ts.do(t, http.MethodGet, "/", cookie, nil); after.Code != http.StatusSeeOther {
		t.Fatalf("session survived logout: status=%d", after.Code)
	}
}

func TestLoginPageRedirectsWhenAlreadyLoggedIn(t *testing.T) {
	ts := newAuthTestServer(t)
	ts.createUser(t, "ana", "clave-segura-1", models.RoleTechnician, false)
	_, cookie := ts.login(t, "ana", "clave-segura-1")

	rec := ts.do(t, http.MethodGet, "/login", cookie, nil)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("GET /login logged in: status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}
