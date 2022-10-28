package event

import (
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/core/common"
	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/postgresql"
)

const DefaultQueryTimeout = 5 * time.Second

func NewEventDb(config config.DbAccess) (*EventDb, error) {
	db, err := postgresql.GetPostgresSqlDb(config)
	if err != nil {
		return nil, err
	}
	eventDb := &EventDb{
		Store:             db,
		eventsChannel:     make(chan blockEvents, 1),
		blockEventChannel: make(chan blockEvents),
	}
	go eventDb.addEventsWorker(common.GetRootContext())
	go eventDb.eventBlockController(common.GetRootContext())

	if err := eventDb.AutoMigrate(); err != nil {
		return nil, err
	}
	return eventDb, nil
}

type EventDb struct {
	dbs.Store
	eventsChannel     chan blockEvents
	blockEventChannel chan blockEvents
}

func (edb *EventDb) BeginTransaction() (*EventDb, error) {
	tx := edb.Store.Get().Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("begin transcation: %v", tx.Error)
	}

	edbTx := EventDb{
		Store: edbTransaction{
			Store: edb,
			tx:    tx,
		},
	}
	return &edbTx, nil
}

func (edb *EventDb) CommitTransaction() error {
	if edb.Store.Get() == nil {
		return errors.New("committing nil transaction")
	}
	return edb.Store.Get().Commit().Error
}

func (edb *EventDb) RollbackTransaction() error {
	if edb.Store.Get() == nil {
		return errors.New("rollbacking nil transaction")
	}
	return edb.Store.Get().Rollback().Error
}

type blockEvents struct {
	block     string
	blockSize int
	round     int64
	events    []Event
	doneC     chan struct{}
}

func (edb *EventDb) AutoMigrate() error {
	if err := edb.Store.Get().AutoMigrate(
		&Event{},
		&Blobber{},
		&User{},
		&Transaction{},
		&WriteMarker{},
		&Validator{},
		&ReadMarker{},
		&Block{},
		&Error{},
		&Miner{},
		&Sharder{},
		&Curator{},
		&DelegatePool{},
		&Allocation{},
		&Reward{},
		&Authorizer{},
		&Challenge{},
		&AllocationBlobberTerm{},
		&ProviderRewards{},
		&ChallengePool{},
	); err != nil {
		return err
	}
	return nil
}
