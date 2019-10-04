package lock

import (
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/log"
)

const LockKind = "Lock"

// Lock is the model used to store locks in the datastore.
type Lock struct {
	Created    time.Time `datastore:"created"`
	Expiration time.Time `datastore:"expiration"`
}

func GetLock(tx *datastore.Transaction, lockOwner string, path string) (*Lock, error) {
	lockKey := datastore.NameKey(LockKind, lockOwner+path, nil)
	var lock Lock
	err := tx.Get(lockKey, &lock)

	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			// Not finding the lock is fine. Just return nil
			return nil, nil
		}
		log.Error(err)
		return nil, err
	}
	return &lock, nil
}

func WriteLock(tx *datastore.Transaction, lockOwner string, path string, expiration time.Time) error {
	lock := &Lock{
		Created:    time.Now(),
		Expiration: expiration,
	}
	lockKey := datastore.NameKey(LockKind, lockOwner+path, nil)
	_, err := tx.Put(lockKey, lock)
	return err
}

func DeleteLock(client *datastore.Client, lockOwner string, path string) error {
	lockKey := datastore.NameKey(LockKind, lockOwner+path, nil)
	return client.Delete(context.Background(), lockKey)
}

func CalculateExpiration(duration int64) time.Time {
	return time.Now().
		Add(time.Duration(duration) * time.Second)
}

// Acquire attempts to acquire a lock for the provided key (owner + path) for
// duration seconds. If the lock already exists and hasn't expired, or if
// the lock cannot be written for unexpected reasons, an error is returned.
func Acquire(datastoreClient *datastore.Client, lockOwner string, path string, duration int64) error {
	_, err := datastoreClient.RunInTransaction(context.Background(), func(tx *datastore.Transaction) error {
		// The output file doesn't exist. Look for the lock
		log.Debug(fmt.Sprintf("looking for lock on %v", path))

		lockObj, err := GetLock(tx, lockOwner, path)
		if err != nil {
			// Unable to get lock. Fail
			log.Debug(err)
			return err
		}

		// The video is currently being generated. Fail this message because we need to keep trying until the output file exists
		if lockObj != nil && lockObj.Expiration.After(time.Now()) {
			return fmt.Errorf("active lock found. expires at %v", lockObj.Expiration)
		}

		// There's no lock, or the lock is expired. Write a lock.
		log.Debug("writing new lock")
		return WriteLock(tx, lockOwner, path, CalculateExpiration(duration))
	})

	return err
}
