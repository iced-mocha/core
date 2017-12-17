package handlers

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
	handler CoreHandler
	manager MockManager
	router  *mux.Router
}

const (
	validRedditAuthJSON = `{"token": "test", "refresh-token": "test", "type": "reddit", "username": "test"}`
	validWeightsJSON    = `{"reddit": 40.0}`
	incompleteAuthJSON  = `{"token": "test", "username": "test"}`
	validUserJSON       = `{"username": "jack", "password": "password"}`
	existsJSON          = `{"username": "exists", "password": "password"}`
	invalidUsernameJSON = `{"username": "s", "password": "password"}`
	invalidPasswordJSON = `{"username": "long", "password": "pas swor  d"}`
)

func (suite *HandlersTestSuite) SetupSuite() {
	// Disable logging while testing
	log.SetOutput(ioutil.Discard)

	manager := &MockManager{}
	m := &MockDriver{}
	suite.handler = CoreHandler{Driver: m, SessionManager: manager}

	// In order to test using path params we need to run a server and send requests to it
	suite.router = mux.NewRouter()
	suite.router.HandleFunc("/v1/users/{userID}/authorize/reddit", suite.handler.UpdateRedditAuth).Methods(http.MethodPost)
	suite.router.HandleFunc("/v1/users/{userID}/weights", suite.handler.UpdateWeights).Methods(http.MethodPost)
	suite.router.HandleFunc("/v1/users/{userID}/accounts/{type}", suite.handler.DeleteLinkedAccount).Methods(http.MethodDelete)
	suite.router.HandleFunc("/v1/users", suite.handler.InsertUser).Methods(http.MethodPost)
}

func addValidSession(r *http.Request) {
	cookie := http.Cookie{Name: testCookie, Value: "valid"}
	r.AddCookie(&cookie)
}

func addInvalidSession(r *http.Request) {
	cookie := http.Cookie{Name: testCookie, Value: "invalid"}
	r.AddCookie(&cookie)
}

func (suite *HandlersTestSuite) TestInsertUser() {
	// Make sure we can get a 200 when sending valid request
	r, err := http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(validUserJSON))
	// NOTE i think this wwont pparse path params correctly
	suite.Nil(err)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure we can get a 400 when inserting a user that already exists
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(existsJSON))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure empty request body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(""))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure invalid username results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(invalidUsernameJSON))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure invalid password results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(invalidPasswordJSON))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure non JSON body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString("\"not json\": \"test\"}"))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *HandlersTestSuite) TestUpdateRedditAuth() {
	// Make sure we can get a 200 when sending valid request
	r, err := http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(validRedditAuthJSON))
	suite.Nil(err)
	addValidSession(r)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure we can get a 401 when sending valid request but dont have a valid session
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(validRedditAuthJSON))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 401 when sending valid request but have an invalid session
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(validRedditAuthJSON))
	suite.Nil(err)
	addInvalidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 403 when sending valid request but are trying to update another accounts info
	r, err = http.NewRequest(http.MethodPost, "/v1/users/user/authorize/reddit", bytes.NewBufferString(validRedditAuthJSON))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusForbidden, w.Code)

	// Make sure sending incomplete set of values results in 400
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(incompleteAuthJSON))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure empty request body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(""))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure non JSON body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/authorize/reddit", bytes.NewBufferString(`"not json"}`))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)
}

func (suite *HandlersTestSuite) TestDeleteLinkedAccount() {
	// Make sure we can get a 200 when sending valid request
	r, err := http.NewRequest(http.MethodDelete, "/v1/users/userID/accounts/reddit", nil)
	suite.Nil(err)
	addValidSession(r)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure we can get a 401 when sending request without a cookie
	r, err = http.NewRequest(http.MethodDelete, "/v1/users/userID/accounts/reddit", nil)
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 401 when sending request with an invalid session
	r, err = http.NewRequest(http.MethodDelete, "/v1/users/userID/accounts/reddit", nil)
	suite.Nil(err)
	addInvalidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 403 when sending request with a session for another user
	r, err = http.NewRequest(http.MethodDelete, "/v1/users/user/accounts/reddit", nil)
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusForbidden, w.Code)

	// Make sure we can get a 400 when sending request with a unrecognized account type
	r, err = http.NewRequest(http.MethodDelete, "/v1/users/userID/accounts/fake", nil)
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

}

func (suite *HandlersTestSuite) TestUpdateWeights() {
	// Make sure we can get a 200 when sending valid request
	r, err := http.NewRequest(http.MethodPost, "/v1/users/userID/weights", bytes.NewBufferString(validWeightsJSON))
	suite.Nil(err)
	addValidSession(r)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure we can get a 401 when sending valid request but dont have any cookie attached to request
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/weights", bytes.NewBufferString(validWeightsJSON))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 401 when sending valid request but have an invalid session
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/weights", bytes.NewBufferString(validWeightsJSON))
	suite.Nil(err)
	addInvalidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusUnauthorized, w.Code)

	// Make sure we can get a 403 when sending valid request but are trying to update another accounts info
	r, err = http.NewRequest(http.MethodPost, "/v1/users/user/weights", bytes.NewBufferString(validWeightsJSON))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusForbidden, w.Code)

	// Make sure empty request body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/weights", bytes.NewBufferString(""))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure non JSON body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users/userID/weights", bytes.NewBufferString(`"not json"}`))
	suite.Nil(err)
	addValidSession(r)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}
