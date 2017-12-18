package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/iced-mocha/core/sessions"
)

const (
	testCookie = "cookie"
)

type MockSession struct {
}

func (m *MockSession) Set(key string, value interface{}) error {
	return nil
}

func (m *MockSession) Get(key string) interface{} {
	if key == "username" {
		return "userID"
	}

	return nil
}

func (m *MockSession) Delete(key string) error {
	return nil
}

func (m *MockSession) SessionID() string {
	return "sid"
}

type MockManager struct {
}

// Mock GetSession function returns nil error if there is a cookie value of 'valid'
func (m *MockManager) GetSession(r *http.Request) (sessions.Session, error) {
	cookie, err := r.Cookie(testCookie)
	if err != nil {
		return nil, err
	}

	value, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return nil, errors.New("unable to get session, likely invalid session id")
	}

	if value == "valid" {
		return &MockSession{}, nil
	}

	return nil, errors.New("error")
}

func (m *MockManager) HasSession(r *http.Request) bool {
	_, err := r.Cookie(testCookie)
	if err != nil {
		return false
	}

	return true
}

func (m *MockManager) SessionStart(w http.ResponseWriter, r *http.Request) sessions.Session {
	cookie := http.Cookie{Name: testCookie, Value: "sid"}
	http.SetCookie(w, &cookie)
	return &MockSession{}
}

func (m *MockManager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
}
