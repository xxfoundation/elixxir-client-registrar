package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"testing"
)

// Storage struct is the API for the storage layer
type Storage struct {
	// Stored Database interface
	database
}

// GetMapImpl is a test use only function for exposing MapImpl
func (s *Storage) GetMapImpl(t *testing.T) *MapImpl {
	return s.database.(*MapImpl)
}

// PopulateClientRegistrationCodes adds Client registration codes to the Database
func (s *Storage) PopulateClientRegistrationCodes(codes []string, uses int) {
	for _, code := range codes {
		err := s.InsertClientRegCode(code, uses)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Client registration code: %+v",
				err)
		}
	}
}
