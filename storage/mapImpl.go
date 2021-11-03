package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// NewMap initializes the database interface with Map backend
func NewMap() Storage {
	defer jww.INFO.Println("Map backend initialized successfully!")
	return Storage{
		&MapImpl{
			clients: make(map[string]*RegistrationCode),
			users:   make(map[string]*User),
			state:   make(map[string]string),
		}}
}

// InsertClientRegCode inserts Client registration code with given number of uses
func (m *MapImpl) InsertClientRegCode(code string, uses int) error {
	m.Lock()
	jww.INFO.Printf("Inserting code %s, %d uses remaining", code, uses)
	// Enforce unique registration code
	if m.clients[code] != nil {
		m.Unlock()
		return errors.Errorf("client registration code %s already exists", code)
	}
	m.clients[code] = &RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	}
	m.Unlock()
	return nil
}

// UseCode if Client registration code is valid, decrements remaining uses
func (m *MapImpl) UseCode(code string) error {
	m.Lock()
	// Look up given registration code
	jww.INFO.Printf("Attempting to use code %s...", code)
	reg := m.clients[code]
	if reg == nil {
		// Unable to find code, return error
		m.Unlock()
		return errors.Errorf("invalid registration code")
	}

	if reg.RemainingUses < 1 {
		// Code has no remaining uses, return error
		m.Unlock()
		return errors.Errorf("registration code %s has no remaining uses", code)
	}

	// Decrement remaining uses by one
	reg.RemainingUses -= 1
	jww.INFO.Printf("Code %s used, %d uses remaining", code,
		reg.RemainingUses)
	m.Unlock()
	return nil
}

// GetUser fetches User from the map
func (m *MapImpl) GetUser(publicKey string) (*User, error) {
	if usr, ok := m.users[publicKey]; ok {
		return &User{
			PublicKey:             publicKey,
			ReceptionKey:          usr.ReceptionKey,
			RegistrationTimestamp: usr.RegistrationTimestamp,
		}, nil
	}
	return nil, errors.New("user does not exist")
}

// InsertUser inserts User into the map
func (m *MapImpl) InsertUser(user *User) error {
	m.users[user.PublicKey] = user
	return nil
}

func (m *MapImpl) UpsertState(key, value string) error {
	m.state[key] = value
	return nil
}

func (m *MapImpl) GetState(key string) (string, error) {
	return m.state[key], nil
}
