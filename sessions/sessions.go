package sessions

type Session interface {
	Set(key string, value interface{}) error //set session value
	Get(key string) interface{}              //get session value
	Delete(key string) error                 //delete session value
	SessionID() string                       //back current sessionID
}
