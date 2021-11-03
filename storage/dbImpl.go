package storage

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"time"
)

// DatabaseImpl struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored Database connection
}

// NewDatabase initializes the database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, dbName, address,
	port string) (Storage, error) {

	var err error
	var db *gorm.DB
	//connect to the Database if the correct information is provided
	if address != "" && port != "" {
		// Create the Database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			address, port, username, dbName)
		// Handle empty Database password
		if len(password) > 0 {
			connectString += fmt.Sprintf(" password=%s", password)
		}
		db, err = gorm.Open(postgres.Open(connectString), &gorm.Config{
			Logger: logger.New(jww.TRACE, logger.Config{LogLevel: logger.Info}),
		})
	}

	// Return the map-backend interface
	// in the event there is a Database error or information is not provided
	if (address == "" || port == "") || err != nil {

		if err != nil {
			jww.WARN.Printf("Unable to initialize Database backend: %+v", err)
		} else {
			jww.WARN.Printf("Database backend connection information not provided")
		}

		return NewMap(), nil
	}

	// Get and configure the internal database ConnPool
	sqlDb, err := db.DB()
	if err != nil {
		return Storage{&DatabaseImpl{}}, errors.Errorf("Unable to configure database connection pool: %+v", err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDb.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	sqlDb.SetMaxOpenConns(50)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be idle.
	sqlDb.SetConnMaxIdleTime(10 * time.Minute)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDb.SetConnMaxLifetime(12 * time.Hour)

	// Initialize the Database schema
	// WARNING: Order is important. Do not change without Database testing
	models := []interface{}{
		&RegistrationCode{}, &User{},
	}
	for _, model := range models {
		err = db.AutoMigrate(model)
		if err != nil {
			return Storage{&DatabaseImpl{}}, err
		}
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{&DatabaseImpl{db: db}}, nil

}

// InsertClientRegCode inserts client registration code with given number of uses
func (d *DatabaseImpl) InsertClientRegCode(code string, uses int) error {
	jww.INFO.Printf("Inserting code %s, %d uses remaining", code, uses)
	return d.db.Create(&RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	}).Error
}

// UseCode decrements reg code uses If client registration code is valid
func (d *DatabaseImpl) UseCode(code string) error {
	// Look up given registration code
	regCode := RegistrationCode{}
	jww.INFO.Printf("Attempting to use code %s...", code)
	err := d.db.First(&regCode, "code = ?", code).Error
	if err != nil {
		// Unable to find code, return error
		return err
	}

	if regCode.RemainingUses < 1 {
		// Code has no remaining uses, return error
		return errors.Errorf("Code %s has no remaining uses", code)
	}

	// Decrement remaining uses by one
	regCode.RemainingUses -= 1
	err = d.db.Save(&regCode).Error
	if err != nil {
		return err
	}

	jww.INFO.Printf("Code %s used, %d uses remaining", code,
		regCode.RemainingUses)
	return nil
}

// GetUser gets User from the Database
func (d *DatabaseImpl) GetUser(publicKey string) (*User, error) {
	user := &User{}
	result := d.db.First(&user, "public_key = ?", publicKey)
	return user, result.Error
}

// InsertUser inserts User into the Database
func (d *DatabaseImpl) InsertUser(user *User) error {
	return d.db.Create(user).Error
}

func (d *DatabaseImpl) UpsertState(key, value string) error {
	s := &State{
		key,
		value,
	}
	return d.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&s).Error
}

func (d *DatabaseImpl) GetState(key string) (string, error) {
	s := &State{}
	return s.Value, d.db.First(&s, "key = ?", key).Error
}
