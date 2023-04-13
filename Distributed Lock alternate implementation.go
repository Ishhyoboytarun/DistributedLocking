package main

import (
	"fmt"
	"time"
)

type DistributedLock struct {
	key       string
	timeout   time.Duration
	owner     string
	heartbeat time.Duration
	stopChan  chan bool
}

func NewDistributedLock(key string, timeout time.Duration, heartbeat time.Duration) *DistributedLock {
	return &DistributedLock{
		key:       key,
		timeout:   timeout,
		owner:     "",
		heartbeat: heartbeat,
		stopChan:  make(chan bool),
	}
}

func (dl *DistributedLock) Acquire() error {
	ticker := time.NewTicker(dl.heartbeat)
	defer ticker.Stop()

	for {
		expiry := time.Now().Add(dl.timeout).UnixNano()
		if acquired, err := dl.tryAcquire(expiry); err == nil && acquired {
			go dl.refreshLock(ticker)
			return nil
		}

		select {
		case <-time.After(time.Millisecond * 100):
			continue
		case <-dl.stopChan:
			return fmt.Errorf("lock acquisition cancelled")
		}
	}
}

func (dl *DistributedLock) Release() error {
	if dl.owner == "" {
		return fmt.Errorf("lock not acquired")
	}

	close(dl.stopChan)
	return nil
}

func (dl *DistributedLock) tryAcquire(expiry int64) (bool, error) {
	response, err := db.SetNX(dl.key, expiry, 0).Result()
	if err != nil {
		return false, err
	}

	if response {
		dl.owner = expiry
		return true, nil
	}

	value, err := db.Get(dl.key).Result()
	if err != nil {
		return false, err
	}

	if time.Now().UnixNano() > stringToTime(value) {
		oldValue, err := db.GetSet(dl.key, expiry).Result()
		if err != nil {
			return false, err
		}

		if oldValue == value {
			dl.owner = expiry
			return true, nil
		}
	}

	return false, nil
}

func (dl *DistributedLock) refreshLock(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			expiry := time.Now().Add(dl.timeout).UnixNano()
			db.Set(dl.key, expiry, 0)
		case <-dl.stopChan:
			return
		}
	}
}

func stringToTime(str string) int64 {
	value, _ := strconv.ParseInt(str, 10, 64)
	return value
}


/*

In this implementation, we use a simple key-value store, represented here by the variable db, 
to provide distributed locking. The NewDistributedLock function initializes a new instance of 
the DistributedLock struct with the provided key, timeout, and heartbeat. The Acquire method 
attempts to acquire the lock by setting the value of the key to the current time plus the lock 
timeout, using the SETNX command to ensure that the key is only set if it does not already exist. 
If the lock is successfully acquired, a background goroutine is started to periodically refresh 
the lock value by updating the value of the key with the current time plus the lock timeout. 
If the lock cannot be acquired, the method waits for a short time before trying again.


The Release method releases the lock by cancelling the background goroutine started by Acquire. 
The tryAcquire method attempts to acquire the lock by checking if the key is available, and 
setting the value of the key if it is.

*/
