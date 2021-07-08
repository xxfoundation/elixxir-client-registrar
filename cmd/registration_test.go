package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client-registrar/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/registration/testkeys"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

var dblck sync.Mutex
var testParams Params
var nodeKey []byte
var permAddr = "0.0.0.0:5900"

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	var err error
	nodeKey, err = utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		fmt.Printf("Could not get node key: %+v\n", err)
	}

	testParams = Params{
		Address:           permAddr,
		CertPath:          testkeys.GetCACertPath(),
		KeyPath:           testkeys.GetCAKeyPath(),
		publicAddress:     permAddr,
		userRegCapacity:   5,
		userRegLeakPeriod: time.Hour,
	}

	runFunc := func() int {
		code := m.Run()
		return code
	}

	os.Exit(runFunc())
}

// Happy Path:  Insert a reg code along with a node
func TestRegCodeExists_RegUser(t *testing.T) {
	dblck.Lock()
	defer dblck.Unlock()
	var err error
	db, _, err := storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	// Initialize an implementation and the permissioning server
	impl, err := StartRegistrar(testParams, &db)
	if err != nil {
		t.Errorf("Unable to start: %+v", err)
	}

	// Insert regcodes into it
	err = db.InsertClientRegCode("AAAA", 100)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}

	// Attempt to register a user
	msg := &pb.UserRegistration{
		RegistrationCode:         "AAAA",
		ClientRSAPubKey:          string(nodeKey),
		ClientReceptionRSAPubKey: string(nodeKey),
	}
	response, err := impl.RegisterUser(msg)

	if err != nil {
		t.Errorf("Failed to register a node when it should have worked: %+v", err)
	}

	if response.ClientReceptionSignedByServer == nil || response.ClientSignedByServer == nil {
		t.Errorf("Failed to sign public key, recieved %+v as a signature & %+v as a receptionSignature",
			response.ClientSignedByServer, response.ClientReceptionSignedByServer)
	}
	impl.Comms.Shutdown()
}

// Happy Path: Inserts users until the max is reached, waits until the timer has
// cleared the number of allowed registrations and inserts another user.
func TestRegCodeExists_RegUser_Timer(t *testing.T) {
	dblck.Lock()
	defer dblck.Unlock()

	// Initialize the database
	db, _, err := storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	testParams2 := Params{
		Address:  "0.0.0.0:5905",
		CertPath: testkeys.GetCACertPath(),
		KeyPath:  testkeys.GetCAKeyPath(),

		publicAddress:     "0.0.0.0:5905",
		userRegCapacity:   4,
		userRegLeakPeriod: 3 * time.Second,
	}

	// Start registration server
	impl, err := StartRegistrar(testParams2, &db)
	if err != nil {
		t.Fatal(err.Error())
	}

	for i := 0; i < int(testParams2.userRegCapacity); i++ {
		// Attempt to register a user
		msg := &pb.UserRegistration{
			RegistrationCode:         "",
			ClientRSAPubKey:          strconv.Itoa(i),
			ClientReceptionRSAPubKey: strconv.Itoa(i),
		}
		_, err = impl.RegisterUser(msg)
		if err != nil {
			t.Errorf("Failed to register a user when it should have worked: %+v", err)
		}

	}

	msg := &pb.UserRegistration{
		RegistrationCode:         "",
		ClientRSAPubKey:          strconv.Itoa(int(testParams2.userRegCapacity)),
		ClientReceptionRSAPubKey: strconv.Itoa(int(testParams2.userRegCapacity)),
	}

	// Attempt to register a user once capacity has been reached
	_, err = impl.RegisterUser(msg)
	if err == nil {
		t.Errorf("Did not fail to register a user when it should not have worked: %+v", err)
	}

	// Attempt to register a user after waiting for capacity to be reset
	time.Sleep(testParams2.userRegLeakPeriod)
	_, err = impl.RegisterUser(msg)
	if err != nil {
		t.Errorf("Failed to register a user when it should have worked: %+v", err)
	}

	impl.Comms.Shutdown()
}
