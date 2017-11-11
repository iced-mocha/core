package sessions

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Manager for managing all sessions within the application
type Manager struct {
	cookieName  string     // Name of the cookie we are storing in the users cookies -- essentially the key of where to look for a session id
	lock        sync.Mutex // Mutex lock to protext unwanted mutation of sessions
	provider    Provider   // Essesntially a storage driver for our sessions
	maxlifetime int64      // An expiry time for our sessions
}

// Various providers that are available to store sessions -- Maps driver names to the actual drivers
// TODO: I dont really like doing it this way -- would rather use dependency injection. FYI this is how the go sql drivers do it
var providers = make(map[string]Provider)

// Register makes a session provider available by the provided name.
func Register(name string, provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider of name %v is nil", name)
	} else if name == "" {
		return errors.New("provided name cannot be empty")
	}

	providers[name] = provider
	return nil
}

// Creates a new session manager based on the given paramaters
func NewManager(providerName, cookieName string, maxlifetime int64) (*Manager, error) {
	provider, ok := providers[providerName]
	if !ok {
		return nil, fmt.Errorf("requested unknown provider. %v not registered", providerName)
	}
	return &Manager{provider: provider, cookieName: cookieName, maxlifetime: maxlifetime}, nil
}

// Session garbage collection -- TODO: If we choose to persist cookies to files this will have to be changed
func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.provider.SessionGC(manager.maxlifetime)
	time.AfterFunc(time.Duration(manager.maxlifetime), func() { manager.GC() })
}

func (manager *Manager) HasSession(r *http.Request) bool {
	cookie, err := r.Cookie(manager.cookieName)
	return (err == nil && cookie.Value != "")
}

// checks the existence of any sessions related to the current user, and creates a new session if none is found.
func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	// Check for a session id store at our cookie name in the incoming requests cookies
	if !manager.HasSession(r) {
		log.Printf("No session cookie found, creating one now")
		// No session id could be found so lets create a new session and store it in the users cookies
		sid := manager.sessionId()
		session, _ = manager.provider.SessionInit(sid)
		// TODO: HttpOnly should probably be true
		cookie := http.Cookie{Name: manager.cookieName, Value: url.QueryEscape(sid), Path: "/", HttpOnly: false, MaxAge: int(manager.maxlifetime)}
		http.SetCookie(w, &cookie)
		log.Printf("Writing cookie for session id: %v", sid)
		return
	}

	// Otherwise the session exists so lets get the id and from that the session

	// We know the cookie exists at this point so ignore the error
	cookie, _ := r.Cookie(manager.cookieName)
	sid, _ := url.QueryUnescape(cookie.Value)
	session, _ = manager.provider.SessionRead(sid)
	return
}

// Destroys the session stored in the requests cookies -- Needs to be called on logout
func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	// Get the session cookie from the request
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		// No cookie to delete so return
		return
	} else {
		manager.lock.Lock()
		defer manager.lock.Unlock()
		manager.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()
		// TODO: Not sure why this needs to be done
		cookie := http.Cookie{Name: manager.cookieName, Path: "/", HttpOnly: true, Expires: expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

// Produces a unique sessionId()
func (manager *Manager) sessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
