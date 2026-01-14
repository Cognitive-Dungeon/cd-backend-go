package session

import "cognitive-server/internal/core/types"

type Session struct {
	objectGuid    types.ObjectGuid
	authenticated bool
}

func NewSession() *Session {
	return &Session{}
}

func (s *Session) Authenticate(guid types.ObjectGuid) {
	s.objectGuid = guid
	s.authenticated = true
}

func (s *Session) IsAuthenticated() bool {
	return s.authenticated
}

func (s *Session) ObjectGuid() types.ObjectGuid {
	return s.objectGuid
}
