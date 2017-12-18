package sessions

import (
	"net/http"
)

type IManager interface {
	GetSession(r *http.Request) (Session, error)

	HasSession(r *http.Request) bool

	SessionStart(w http.ResponseWriter, r *http.Request) Session

	SessionDestroy(w http.ResponseWriter, r *http.Request)
}
