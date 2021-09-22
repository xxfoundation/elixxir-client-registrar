package cmd

import (
	"crypto/x509"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client-registrar/storage"
	"gitlab.com/elixxir/comms/clientregistrar"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"gitlab.com/xx_network/primitives/utils"
)

type Impl struct {
	Comms        *clientregistrar.Comms
	rl           *rateLimiting.Bucket
	pk           *rsa.PrivateKey
	cert         *x509.Certificate
	DB           *storage.Storage
	certFromFile []byte
	Stopped      *uint32
}

func StartRegistrar(params Params, db *storage.Storage) (*Impl, error) {
	rsaKeyPem, err := utils.ReadFile(params.KeyPath)
	if err != nil {
		return nil, errors.Errorf("failed to read key at %+v: %+v",
			params.KeyPath, err)
	}

	rsaPrivateKey, err := rsa.LoadPrivateKeyFromPem(rsaKeyPem)
	if err != nil {
		return nil, errors.Errorf("Failed to parse client registrar server key: %+v. "+
			"Registrar key is %+v", err, rsaPrivateKey)
	}

	certFromFile, err := utils.ReadFile(params.CertPath)
	if err != nil {
		return nil, errors.Errorf("failed to read certificate at %+v: %+v", params.CertPath, err)
	}

	// Set globals for permissioning server
	cert, err := tls.LoadCertificate(string(certFromFile))
	if err != nil {
		return nil, errors.Errorf("Failed to parse client registrar server cert: %+v. "+
			"Registrar cert is %+v", err, certFromFile)
	}

	impl := &Impl{
		pk:           rsaPrivateKey,
		cert:         cert,
		certFromFile: certFromFile,
		DB:           db,
	}
	// TODO: ID for client registrar
	impl.Comms = clientregistrar.StartClientRegistrarServer(&id.Permissioning, params.Address, NewImplementation(impl), certFromFile, rsaKeyPem)
	impl.rl = rateLimiting.CreateBucket(params.userRegCapacity, params.userRegCapacity, params.userRegLeakPeriod, func(u uint32, i int64) {})

	return impl, nil
}

func NewImplementation(instance *Impl) *clientregistrar.Implementation {
	impl := clientregistrar.NewImplementation()
	impl.Functions.RegisterUser = func(msg *pb.ClientRegistration) (*pb.SignedClientRegistrationConfirmations, error) {
		confirmationMessage, err := instance.RegisterUser(msg)
		if err != nil {
			jww.ERROR.Printf("RegisterUser error: %+v", err)
		}
		return confirmationMessage, err
	}
	return impl
}
