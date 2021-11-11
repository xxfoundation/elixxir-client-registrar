package storage

import (
	"sync"
	"time"
)

var (
	BucketUserRegCapacityKey   = "BucketCapacity"
	BucketUserRegLeakPeriodKey = "BucketPeriod"
)

// database interface defines base methods for storage
type database interface {
	InsertClientRegCode(code string, uses int) error
	UseCode(code string) error
	GetUser(publicKey string) (*User, error)
	InsertUser(user *User) error
	UpsertState(key, value string) error
	GetState(key string) (string, error)
}

// MapImpl struct is intended to mock the behavior of a real database
type MapImpl struct {
	clients map[string]*RegistrationCode
	users   map[string]*User
	state   map[string]string
	sync.Mutex
}

// Struct representing a RegistrationCode table in the Database
type RegistrationCode struct {
	// Registration code acts as the primary key
	Code string `gorm:"primary_key"`
	// Remaining uses for the RegistrationCode
	RemainingUses int
}

// Struct representing the User table in the Database
type User struct {
	// User TLS public certificate in PEM string format
	PublicKey string `gorm:"primary_key"`
	// User reception key in PEM string format
	ReceptionKey string `gorm:"NOT NULL;UNIQUE"`
	// Timestamp in which user registered with permissioning
	RegistrationTimestamp time.Time `gorm:"NOT NULL"`
}

type RegistrarState struct {
	Key   string `gorm:"primary_key"`
	Value string `gorm:"NOT NULL"`
}
