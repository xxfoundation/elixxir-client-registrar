package storage

import (
	"testing"
	"time"
)

//func TestDatabaseImpl(t *testing.T) {
//	db, _, err := NewDatabase("postgres", "", "clientregistrar", "0.0.0.0", "5432")
//	if err != nil {
//		t.Errorf("Failed to create DB: %+v", err)
//		t.FailNow()
//	}
//	err = db.PopulateClientRegistrationCodes([]string{"AAAAA", "BBBBB", "CCCCC"}, 1)
//	if err != nil {
//		t.Errorf("Failed to populate reg codes: %+v", err)
//	}
//
//	err = db.UseCode("AAAAA")
//	if err != nil {
//		t.Errorf("Failed to use code AAAAA: %+v", err)
//	}
//
//	err = db.UseCode("AAAAA")
//	if err == nil {
//		t.Errorf("Should not have been able to use code AAAAA again")
//	}
//
//	err = db.InsertUser(&User{
//		PublicKey:             "pub",
//		ReceptionKey:          "reception",
//		RegistrationTimestamp: time.Now(),
//	})
//	if err != nil {
//		t.Errorf("Failed to insert user: %+v", err)
//	}
//	u, err := db.GetUser("pub")
//	if err != nil {
//		t.Errorf("Failed to get user: %+v", err)
//	}
//	if u.ReceptionKey != "reception" {
//		t.Errorf("Wow somehow you got the wrong user back this shouldn't happen")
//	}
//}

// Happy path
func TestMapImpl_InsertClientRegCode(t *testing.T) {
	m := &MapImpl{
		clients: make(map[string]*RegistrationCode),
	}

	// Attempt to load in a valid code
	code := "TEST"
	uses := 100
	err := m.InsertClientRegCode(code, uses)

	// Verify the insert was successful
	if err != nil || m.clients[code] == nil || m.clients[code].
		RemainingUses != uses {
		t.Errorf("Expected to successfully insert client registration code")
	}
}

// Error Path: Duplicate client registration code
func TestMapImpl_InsertClientRegCode_Duplicate(t *testing.T) {
	m := &MapImpl{
		clients: make(map[string]*RegistrationCode),
	}

	// Load in a registration code
	code := "TEST"
	uses := 100
	m.clients[code] = &RegistrationCode{Code: code}

	// Attempt to load in a duplicate code
	err := m.InsertClientRegCode(code, uses)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate client registration" +
			" code")
	}
}

// Happy path
func TestMapImpl_UseCode(t *testing.T) {
	m := &MapImpl{
		clients: make(map[string]*RegistrationCode),
	}

	// Load in a registration code
	code := "TEST"
	uses := 100
	m.clients[code] = &RegistrationCode{Code: code, RemainingUses: uses}

	// Verify the code was used successfully
	err := m.UseCode(code)
	if err != nil || m.clients[code].RemainingUses != uses-1 {
		t.Errorf("Expected using client registration code to succeed")
	}
}

// Error Path: No remaining uses of client registration code
func TestMapImpl_UseCode_NoRemainingUses(t *testing.T) {
	m := &MapImpl{
		clients: make(map[string]*RegistrationCode),
	}

	// Load in a registration code
	code := "TEST"
	uses := 0
	m.clients[code] = &RegistrationCode{Code: code, RemainingUses: uses}

	// Verify the code was used successfully
	err := m.UseCode(code)
	if err == nil {
		t.Errorf("Expected using client registration code with no remaining" +
			" uses to fail")
	}
}

// Error Path: Invalid client registration code
func TestMapImpl_UseCode_Invalid(t *testing.T) {
	m := &MapImpl{
		clients: make(map[string]*RegistrationCode),
	}

	// Verify the code was used successfully
	err := m.UseCode("TEST")
	if err == nil {
		t.Errorf("Expected using invalid client registration code with no to" +
			" fail")
	}
}

// Happy path
func TestMapImpl_InsertUser(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]*User),
	}

	testKey := "TEST"
	_ = m.InsertUser(&User{
		PublicKey:             testKey,
		ReceptionKey:          testKey,
		RegistrationTimestamp: time.Now(),
	})
	if _, ok := m.users[testKey]; !ok {
		t.Errorf("Insert failed to add the user!")
	}
}

// Happy path
func TestMapImpl_GetUser(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]*User),
	}

	testKey := "TEST"
	m.users[testKey] = &User{
		PublicKey: testKey,
	}

	user, err := m.GetUser(testKey)
	if err != nil || user.PublicKey != testKey {
		t.Errorf("Get failed to get user!")
	}
}

// Get user that does not exist
func TestMapImpl_GetUserNotExists(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]*User),
	}

	testKey := "TEST"

	_, err := m.GetUser(testKey)
	if err == nil {
		t.Errorf("Get expected to not find user!")
	}
}

func TestMapImpl_GetState(t *testing.T) {
	m := &MapImpl{state: make(map[string]string)}
	testVal := "i'm a value"
	m.state[BucketUserRegLeakPeriodKey] = testVal
	val, err := m.GetState(BucketUserRegLeakPeriodKey)
	if err != nil {
		t.Errorf("Failed to get state value %s: %+v", BucketUserRegLeakPeriodKey, err)
	} else if val != testVal {
		t.Errorf("Expected key '%s' to return '%s' but got '%s' instead", BucketUserRegLeakPeriodKey, testVal, val)
	}

	val, err = m.GetState(BucketUserRegCapacityKey)
	if err == nil {
		t.Errorf("Expected an error, but did not receive one")
	}
}

func TestMapImpl_UpsertState(t *testing.T) {
	m := &MapImpl{state: make(map[string]string)}
	testVal := "i'm a value"
	err := m.UpsertState(BucketUserRegCapacityKey, testVal)
	if err != nil {
		t.Errorf("Failed to upsert state: %+v", err)
	}
	if m.state[BucketUserRegCapacityKey] != testVal {
		t.Errorf("Failed to properly upsert state")
	}
}
