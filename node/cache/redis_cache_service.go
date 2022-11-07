package cache

import (
	"context"
	"runtime"
	"strings"

	"github.com/go-redis/redis/v8"

	"golang.org/x/xerrors"
)

type RedisCacheSvc struct {
	Ctx    context.Context
	Client redis.Cmdable
}

var (
	redisCacheSvc *RedisCacheSvc
)

func NewRedisCacheSvc(conn string, password string, poolSize int) *RedisCacheSvc {
	once.Do(func() {
		log.Infof("octopus: init redis client: %v ******", conn)

		if poolSize < 1 {
			poolSize = 4 * runtime.NumCPU()
		}
		var cli redis.Cmdable
		if strings.Contains(conn, ",") {
			cli = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:    strings.Split(conn, ","),
				Password: password,
				PoolSize: poolSize,
			})
		} else {
			cli = redis.NewClient(&redis.Options{
				Addr:     conn,
				Password: password,
				PoolSize: poolSize,
			})
		}

		if cli != nil {
			redisCacheSvc = &RedisCacheSvc{
				Client: cli,
				Ctx:    context.Background(),
			}
		}
	})
	return redisCacheSvc
}

func (svc *RedisCacheSvc) CreateCache(name string, capacity int) error {
	return nil
}

func (svc *RedisCacheSvc) Get(name string, key string) (interface{}, error) {
	exists, err := svc.Client.Exists(svc.Ctx, name+"_"+key).Result()
	if err != nil {
		return nil, err
	}
	if exists == 1 {
		value, err := svc.Client.Get(svc.Ctx, name+"_"+key).Result()
		if err != nil {
			return nil, err
		}
		return value, nil
	}

	return nil, xerrors.Errorf("not found")
}

func (svc *RedisCacheSvc) Put(name string, key string, value interface{}) {
	_, err := svc.Client.Set(svc.Ctx, key, value, 0).Result()
	if err != nil {
		log.Error(err.Error())
	}
}

func (svc *RedisCacheSvc) Evict(name string, key string) {
	_, err := svc.Client.Del(svc.Ctx, key, name+"_"+key).Result()
	if err != nil {
		log.Error(err.Error())
	}
}

func (svc *RedisCacheSvc) GetCapacity(name string) int {
	log.Warn("depends on redis capacity")

	return -1
}

func (svc *RedisCacheSvc) GetSize(name string) int {
	log.Warn("depends on redis capacity")

	return -1
}

func (svc *RedisCacheSvc) ReSize(name string, capacity int) error {
	log.Warn("unsupport operation, depends on redis capacity")

	return nil
}
