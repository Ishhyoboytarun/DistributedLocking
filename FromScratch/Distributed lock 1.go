package FromScratch

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type DistributedLock struct {
	client *clientv3.Client
	key    string
	val    string
	lease  clientv3.LeaseID
}

func NewDistributedLock(endpoints []string, key, val string) (*DistributedLock, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &DistributedLock{
		client: client,
		key:    key,
		val:    val,
	}, nil
}

func (l *DistributedLock) Lock() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := l.client.Grant(ctx, 10)
	if err != nil {
		return err
	}

	_, err = l.client.Put(ctx, l.key, l.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}

	ch, err := l.client.KeepAlive(ctx, resp.ID)
	if err != nil {
		return err
	}

	go func() {
		for range ch {
		}
	}()

	l.lease = resp.ID
	return nil
}

func (l *DistributedLock) Unlock() error {
	_, err := l.client.Revoke(context.Background(), l.lease)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	endpoints := []string{"localhost:2379"}
	key := "/distributed-lock"
	val := "example-lock-value"

	lock, err := NewDistributedLock(endpoints, key, val)
	if err != nil {
		log.Fatal(err)
	}

	err = lock.Lock()
	if err != nil {
		log.Fatal(err)
	}

	// Do some critical section work here

	err = lock.Unlock()
	if err != nil {
		log.Fatal(err)
	}
}
