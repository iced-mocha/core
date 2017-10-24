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
	router  *mux.Router
}

const (
	redditAuthJSON = "{\"bearertoken\": \"test\"}"
	userJSON       = "{\"name\": \"jack\"}"
)

func (suite *HandlersTestSuite) SetupSuite() {
	// Disable logging while testing
	log.SetOutput(ioutil.Discard)

	suite.handler = CoreHandler{Driver: &MockDriver{}}

	// In order to test using path params we need to run a server and send requests to it
	suite.router = mux.NewRouter()
	suite.router.HandleFunc("/v1/user/{userID}/authorize/reddit", suite.handler.UpdateRedditAuth).Methods(http.MethodPost)
	suite.router.HandleFunc("/v1/users", suite.handler.InsertUser).Methods(http.MethodPost)
}

func (suite *HandlersTestSuite) TestInsertUser() {
	// Make sure we can get a 200 when sending valid request
	r, err := http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(userJSON))
	suite.Nil(err)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure empty request body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/users", bytes.NewBufferString(""))
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
	r, err := http.NewRequest(http.MethodPost, "/v1/user/userID/authorize/reddit", bytes.NewBufferString(redditAuthJSON))
	suite.Nil(err)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusOK, w.Code)

	// Make sure empty request body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/user/userID/authorize/reddit", bytes.NewBufferString(""))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

	// Make sure non JSON body results in 400 bad request
	r, err = http.NewRequest(http.MethodPost, "/v1/user/userID/authorize/reddit", bytes.NewBufferString("\"not json\"}"))
	suite.Nil(err)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, r)
	suite.Equal(http.StatusBadRequest, w.Code)

}

func TestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}