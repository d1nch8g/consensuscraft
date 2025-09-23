package database

import (
	"errors"
	"sync"
	"time"

	"github.com/d1nch8g/consensuscraft/gen/pb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type ChangeEntry struct {
	key       []byte
	value     []byte
	timestamp time.Time
	deleted   bool
}

type DB struct {
	leveldb   *leveldb.DB
	mu        sync.RWMutex
	changeLog []ChangeEntry
	closed    bool
}

var ErrClosed = errors.New("database is closed")

func New(path string) (*DB, error) {
	ldb, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &DB{
		leveldb:   ldb,
		changeLog: make([]ChangeEntry, 0),
	}, nil
}

func (db *DB) Put(key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}

	err := db.leveldb.Put(key, value, nil)
	if err != nil {
		return err
	}

	// Log change for concurrent streaming
	db.changeLog = append(db.changeLog, ChangeEntry{
		key:       append([]byte(nil), key...),
		value:     append([]byte(nil), value...),
		timestamp: time.Now(),
		deleted:   false,
	})

	// Keep change log bounded (last 1000 entries)
	if len(db.changeLog) > 1000 {
		db.changeLog = db.changeLog[len(db.changeLog)-1000:]
	}

	return nil
}

func (db *DB) Get(key []byte) ([]byte, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil, ErrClosed
	}

	return db.leveldb.Get(key, nil)
}

func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return ErrClosed
	}

	err := db.leveldb.Delete(key, nil)
	if err != nil {
		return err
	}

	// Log deletion for concurrent streaming
	db.changeLog = append(db.changeLog, ChangeEntry{
		key:       append([]byte(nil), key...),
		value:     nil,
		timestamp: time.Now(),
		deleted:   true,
	})

	// Keep change log bounded
	if len(db.changeLog) > 1000 {
		db.changeLog = db.changeLog[len(db.changeLog)-1000:]
	}

	return nil
}

func (db *DB) StreamAll() <-chan *pb.SyncDatabaseData {
	ch := make(chan *pb.SyncDatabaseData, 100)

	go func() {
		defer close(ch)

		// Mark sync start point
		syncStart := time.Now()

		// Take snapshot for consistent read
		snapshot, err := db.leveldb.GetSnapshot()
		if err != nil {
			return
		}
		defer snapshot.Release()

		// Stream all snapshot data
		iter := snapshot.NewIterator(util.BytesPrefix(nil), nil)
		defer iter.Release()

		for iter.Next() {
			// Copy data to avoid reference issues
			key := append([]byte(nil), iter.Key()...)
			value := append([]byte(nil), iter.Value()...)

			select {
			case ch <- &pb.SyncDatabaseData{
				Key:   key,
				Value: value,
			}:
			default:
				// Channel full, continue but note potential data loss
				continue
			}
		}

		if err := iter.Error(); err != nil {
			return
		}

		// Stream changes that happened during snapshot read
		db.mu.RLock()
		for _, change := range db.changeLog {
			if change.timestamp.After(syncStart) {
				if change.deleted {
					// Send deletion marker (empty value)
					select {
					case ch <- &pb.SyncDatabaseData{
						Key:   change.key,
						Value: nil,
					}:
					default:
						continue
					}
				} else {
					select {
					case ch <- &pb.SyncDatabaseData{
						Key:   change.key,
						Value: change.value,
					}:
					default:
						continue
					}
				}
			}
		}
		db.mu.RUnlock()
	}()

	return ch
}

func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closed {
		return nil
	}

	db.closed = true
	return db.leveldb.Close()
}

func (db *DB) NewIterator() iterator.Iterator {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.closed {
		return nil
	}

	return db.leveldb.NewIterator(util.BytesPrefix(nil), nil)
}
