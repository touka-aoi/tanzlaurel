package domain

type SessionRegistry interface {
	Add(sessionID SessionID)
	Remove(sessionID SessionID)
	List() []SessionID
	Contains(sessionID SessionID) bool
}

type LocalSessionRegistry struct {
	sessions map[SessionID]struct{}
}

func NewLocalSessionRegistry() *LocalSessionRegistry {
	return &LocalSessionRegistry{
		sessions: make(map[SessionID]struct{}),
	}
}

func (r *LocalSessionRegistry) Add(sessionID SessionID) {
	r.sessions[sessionID] = struct{}{}
}

func (r *LocalSessionRegistry) Remove(sessionID SessionID) {
	delete(r.sessions, sessionID)
}

func (r *LocalSessionRegistry) List() []SessionID {
	list := make([]SessionID, 0, len(r.sessions))
	for id := range r.sessions {
		list = append(list, id)
	}
	return list
}

func (r *LocalSessionRegistry) Contains(sessionID SessionID) bool {
	_, ok := r.sessions[sessionID]
	return ok
}
