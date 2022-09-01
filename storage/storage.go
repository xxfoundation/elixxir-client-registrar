////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"github.com/pkg/errors"
	"strconv"
	"testing"
	"time"
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
func (s *Storage) PopulateClientRegistrationCodes(codes []string, uses int) error {
	for _, code := range codes {
		err := s.InsertClientRegCode(code, uses)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) GetBucketParameters() (uint32, time.Duration, error) {
	scapacity, err := s.GetState(BucketUserRegCapacityKey)
	if err != nil {
		return 0, -1, errors.WithMessage(err, "Failed to get reg bucket capactiy")
	}
	speriod, err := s.GetState(BucketUserRegLeakPeriodKey)
	if err != nil {
		return 0, -1, errors.WithMessage(err, "Failed to get reg bucket leak period")
	}
	capacity, err := strconv.Atoi(scapacity)
	if err != nil {
		return 0, -1, errors.WithMessage(err, "Failed to parse capacity")
	}
	period, err := time.ParseDuration(speriod)
	if err != nil {
		return 0, -1, errors.WithMessage(err, "Failed to parse period")
	}
	return uint32(capacity), period, nil
}

func (s *Storage) UpdateBucketParameters(capacity uint32, period time.Duration) error {
	err := s.UpsertState(BucketUserRegCapacityKey, strconv.Itoa(int(capacity)))
	if err != nil {
		return errors.WithMessage(err, "Failed to upsert bucket reg capacity")
	}
	err = s.UpsertState(BucketUserRegLeakPeriodKey, period.String())
	if err != nil {
		return errors.WithMessage(err, "Failed to upsert bucket leak period")
	}
	return nil
}
