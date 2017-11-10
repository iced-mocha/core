package memory

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/iced-mocha/core/sessions"
)

var provider = &MemoryProvider{list: list.New()}

// Implementation of the Session interface -- no conflict due to being in different packages
type Session struct {
	sid          string                 // unique session id
	timeAccessed time.Time              // last access time
	values       map[string]interface{} // session value stored inside
}

// Set a value in the session store
func (s *Session) Set(key string, value interface{}) error {
	s.values[key] = value
	provider.SessionUpdate(s.sid)
	return nil
}

// TODO: Change signature to be (interface{}, error) ?
func (s *Session) Get(key string) interface{} {
	// Update our session access time
	provider.SessionUpdate(s.sid)
	if v, ok := s.values[key]; ok {
		return v
	}
	return nil
}

func (s *Session) Delete(key string) error {
	// Update our session access time
	provider.SessionUpdate(s.sid)
	delete(s.values, key)
	return nil
}

// Get the id of a given session
func (s *Session) SessionID() string {
	return s.sid
}

type MemoryProvider struct {
	lock     sync.Mutex               // lock
	sessions map[string]*list.Element // save in memory -- maps session ids to session objects (wrapped in list.Element)
	list     *list.List               // gc -- TODO: Figure out what this is for? -- Pretty sure its for easily checking the oldest elements
}

// Creates a new session and stores it in our provider
func (p *MemoryProvider) SessionInit(sid string) (sessions.Session, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	v := make(map[string]interface{}, 0)
	// Create the session, setting the accessed time to now
	s := &Session{sid: sid, timeAccessed: time.Now(), values: v}

	// Wrap the session in a list element -- TODO: Maybe put it on the front?
	element := p.list.PushBack(s)
	// Insert the element into our hash of sessions
	p.sessions[sid] = element
	return s, nil
}

// Produces the session associated with the given session id
func (p *MemoryProvider) SessionRead(sid string) (sessions.Session, error) {
	if v, ok := p.sessions[sid]; ok {
		// The sesson exists so return it
		return v.Value.(*Session), nil
	}

	// No such session so produce an error
	return nil, fmt.Errorf("no such session %v", sid)
}

// Destroys the session associated with the given id
func (p *MemoryProvider) SessionDestroy(sid string) error {
	// Need to check if it exists to remove it from our garbage collection list
	if v, ok := p.sessions[sid]; ok {
		delete(p.sessions, sid)
		p.list.Remove(v)
	}
	return nil
}

// TODO: this function works... but with the way sessions are inserted into the GC list it will not
// Sessions are put into the back of the list when they are created -- not a breaking bug... but will likely
// Result in eventaully having many stale sessions accumulating
func (p *MemoryProvider) SessionGC(maxlifetime int64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for {
		// Get the last element of our GC list
		element := p.list.Back()

		// No more sessions to delete
		if element == nil {
			break
		}
		// TODO: make this check into an `isExpired() function`
		// Checks if the accessed time is older than our maximum allowed lifespan
		if (element.Value.(*Session).timeAccessed.Unix() + maxlifetime) < time.Now().Unix() {
			// So remove it from our GC list
			p.list.Remove(element)
			// Delete it from our stored sessions
			delete(p.sessions, element.Value.(*Session).sid)
		} else {
			// Otherwise we reach the point of the list where they have all
			// been accessed recently enough to not require to be deleted
			break
		}
	}
}

// Updates the session access time to time.Now() and moves it to the front of the list (MTF heuristic)
func (p *MemoryProvider) SessionUpdate(sid string) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if v, ok := p.sessions[sid]; ok {
		// TODO: currently this will crash if its not so we must do something like `val, ok := v.Value.(*SessionStore)`
		// Update the session time to now
		v.Value.(*Session).timeAccessed = time.Now()
		// Uses the move to front heuristic to speed up acess times
		p.list.MoveToFront(v)
		return nil
	}
	return fmt.Errorf("No such session: %v", sid)
}

func init() {
	provider.sessions = make(map[string]*list.Element, 0)
	// Register our memory storage provider
	sessions.Register("memory", provider)
}
