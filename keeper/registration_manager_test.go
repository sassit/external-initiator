package keeper

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/external-initiator/store"
	"github.com/stretchr/testify/require"
)

var address = common.HexToAddress("0x0000000000000000000000000000000000000123")
var checkGasLimit = int64(10_000)

func TestRegistrationManager_PerformFullSync(t *testing.T) {
	db, cleanup := store.SetupTestDB(t)
	defer cleanup()

	assertRegistrationCount(t, db, 0)

	rm := NewRegistrationManager(db)
	rm.PerformFullSync()
	// TODO - add client mocks

	assertRegistrationCount(t, db, 3)
}

func TestRegistrationManager_Upsert(t *testing.T) {
	db, cleanup := store.SetupTestDB(t)
	defer cleanup()

	assertRegistrationCount(t, db, 0)
	rm := NewRegistrationManager(db)

	// create registration
	newRegistration := upkeepRegistration{
		UpkeepID:      0,
		Address:       address,
		CheckGasLimit: checkGasLimit,
	}
	err := rm.Upsert(newRegistration)
	require.NoError(t, err)

	assertRegistrationCount(t, db, 1)
	var existingRegistration upkeepRegistration
	err = db.DB().First(&existingRegistration).Error
	require.NoError(t, err)
	require.Equal(t, checkGasLimit, existingRegistration.CheckGasLimit)
	require.Equal(t, int64(0), existingRegistration.LastRunBlockHeight)

	// update registration
	updatedRegistration := upkeepRegistration{
		UpkeepID:           0,
		Address:            address,
		CheckGasLimit:      20_000,
		LastRunBlockHeight: 100,
	}
	err = rm.Upsert(updatedRegistration)
	require.NoError(t, err)
	assertRegistrationCount(t, db, 1)
	err = db.DB().First(&existingRegistration).Error
	require.NoError(t, err)
	require.Equal(t, int64(20_000), existingRegistration.CheckGasLimit)
	require.Equal(t, int64(100), existingRegistration.LastRunBlockHeight)
}

func TestRegistrationManager_Delete(t *testing.T) {
	db, cleanup := store.SetupTestDB(t)
	defer cleanup()

	rm := NewRegistrationManager(db)

	assertRegistrationCount(t, db, 0)

	// create registration
	registration := upkeepRegistration{
		UpkeepID:      0,
		Address:       address,
		CheckGasLimit: checkGasLimit,
	}
	err := db.DB().Create(&registration).Error
	require.NoError(t, err)
	assertRegistrationCount(t, db, 1)

	// delete
	err = rm.Delete(registration.Address, 0)
	require.NoError(t, err)
	assertRegistrationCount(t, db, 0)

	// delete again
	err = rm.Delete(registration.Address, 0)
	require.NoError(t, err)
	assertRegistrationCount(t, db, 0)

	// delete a non-existent registration
	err = rm.Delete(registration.Address, 1234)
	require.NoError(t, err)
	assertRegistrationCount(t, db, 0)
}

func TestRegistrationManager_BatchDelete(t *testing.T) {
	db, cleanup := store.SetupTestDB(t)
	defer cleanup()

	assertRegistrationCount(t, db, 0)
	address := common.HexToAddress("0x0000000000000000000000000000000000000123")

	registrations := [3]upkeepRegistration{
		{
			UpkeepID:      0,
			Address:       address,
			CheckGasLimit: checkGasLimit,
		}, {
			UpkeepID:      1,
			Address:       address,
			CheckGasLimit: checkGasLimit,
		}, {
			UpkeepID:      2,
			Address:       address,
			CheckGasLimit: checkGasLimit,
		},
	}

	for _, reg := range registrations {
		err := db.DB().Create(&reg).Error
		require.NoError(t, err)
	}

	assertRegistrationCount(t, db, 3)

	rm := NewRegistrationManager(db)
	err := rm.BatchDelete(address, []int64{0, 2})
	require.NoError(t, err)

	assertRegistrationCount(t, db, 1)
}

func TestRegistrationManager_Active(t *testing.T) {
	db, cleanup := store.SetupTestDB(t)
	defer cleanup()

	assertRegistrationCount(t, db, 0)

	// TODO
	// currentBlock := 10
	// coolDown := 3
	address := common.HexToAddress("0x0000000000000000000000000000000000000123")

	// valid
	registration1 := upkeepRegistration{
		UpkeepID:           0,
		Address:            address,
		LastRunBlockHeight: 0, // 0 means never
		CheckGasLimit:      checkGasLimit,
	}
	// upkeep too recent
	registration2 := upkeepRegistration{
		UpkeepID:           1,
		Address:            address,
		LastRunBlockHeight: 7,
		CheckGasLimit:      checkGasLimit,
	}

	for _, reg := range []upkeepRegistration{registration1, registration2} {
		err := db.DB().Create(&reg).Error
		require.NoError(t, err)
	}

	assertRegistrationCount(t, db, 2)

	rm := NewRegistrationManager(db)
	activeRegistrations, err := rm.Active()
	require.NoError(t, err)
	require.Len(t, activeRegistrations, 1)
	require.Equal(t, *big.NewInt(1), activeRegistrations[0].UpkeepID)
}

func assertRegistrationCount(t *testing.T, db *store.Client, expected int) {
	var count int
	db.DB().Model(&upkeepRegistration{}).Count(&count)
	require.Equal(t, expected, count)
}
