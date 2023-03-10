////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"time"
)

// DatabaseImpl struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored Database connection
}

// NewDatabase initializes the database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {

	var err error
	var db *gorm.DB
	//connect to the Database if the correct information is provided
	if address != "" && port != "" {
		// Create the Database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			address, port, username, database)
		// Handle empty Database password
		if len(password) > 0 {
			connectString += fmt.Sprintf(" password=%s", password)
		}
		db, err = gorm.Open("postgres", connectString)
	}

	// Return the map-backend interface
	// in the event there is a Database error or information is not provided
	if (address == "" || port == "") || err != nil {

		if err != nil {
			jww.WARN.Printf("Unable to initialize Database backend: %+v", err)
		} else {
			jww.WARN.Printf("Database backend connection information not provided")
		}

		return NewMap(), func() error { return nil }, nil
	}

	// Initialize the Database logger
	db.SetLogger(jww.TRACE)
	db.LogMode(true)

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	db.DB().SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	db.DB().SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	db.DB().SetConnMaxLifetime(24 * time.Hour)

	// Initialize the Database schema
	// WARNING: Order is important. Do not change without Database testing
	models := []interface{}{
		&RegistrationCode{}, &User{},
	}
	for _, model := range models {
		err = db.AutoMigrate(model).Error
		if err != nil {
			return Storage{}, func() error { return nil }, err
		}
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{&DatabaseImpl{db: db}}, db.Close, nil

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
	s := RegistrarState{
		Key:   key,
		Value: value,
	}
	if err := d.db.Model(&s).Where("key = ?", key).Update("value", value).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return d.db.Create(&s).Error
		}
		return err
	}
	return nil
}

func (d *DatabaseImpl) GetState(key string) (string, error) {
	s := &RegistrarState{}
	return s.Value, d.db.Find(&s, "key = ?", key).Error
}
