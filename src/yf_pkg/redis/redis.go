package redis

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/garyburd/redigo/redis"
)

type BasePool struct {
	BDPool  map[int]*redis.Pool
	Address string
}

type RedisPool struct {
	rPools        []BasePool
	allRPools     []BasePool
	wPool         BasePool
	maxActiveConn int
}

var ErrNil = redis.ErrNil

func newPool(server string, maxActiveConn int, db int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   maxActiveConn,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server, redis.DialDatabase(db))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func New(wAddress string, rAddresses []string, maxActiveConn int) (rp *RedisPool) {
	rPools := make([]BasePool, len(rAddresses))
	allRPools := make([]BasePool, len(rAddresses))
	for i := 0; i < len(rPools); i++ {
		rPools[i] = BasePool{map[int]*redis.Pool{}, rAddresses[i]}
		allRPools[i] = rPools[i]
	}
	wPool := BasePool{map[int]*redis.Pool{}, wAddress}
	rp = &RedisPool{rPools, allRPools, wPool, maxActiveConn}
	go rp.checkAlive()
	return
}

//仅检查从库是否健康
func (rp *RedisPool) checkAlive() {
	for {
		time.Sleep(3 * time.Second)
		tmpPools := make([]BasePool, 0, len(rp.allRPools))
		for _, poolGroup := range rp.allRPools {
			alive := true
			for _, pool := range poolGroup.BDPool {
				conn := pool.Get()
				if _, err := conn.Do("PING"); err != nil {
					fmt.Println(err.Error())
					alive = false
				}
				conn.Close()
				break
			}
			if alive {
				tmpPools = append(tmpPools, poolGroup)
				/*
					pool, ok := pools[dbid]
					if ok {
						fmt.Printf("%v[online]\n", pool.Server)
					}
				*/
			} else {
				fmt.Printf("%v[down] kick\n", poolGroup.Address)
			}
		}
		rp.rPools = tmpPools
	}
}

func (rp *RedisPool) GetReadConnection(db int) redis.Conn {
	var pool *redis.Pool
	var ok bool
	var selected int
	if len(rp.rPools) == 0 {
		selected = rand.Int() % len(rp.allRPools)
		pool, ok = rp.allRPools[selected].BDPool[db]
		if !ok {
			rp.allRPools[selected].BDPool[db] = newPool(rp.allRPools[selected].Address, rp.maxActiveConn, db)
			pool = rp.allRPools[selected].BDPool[db]
		}
	} else {
		selected = rand.Int() % len(rp.rPools)
		pool, ok = rp.rPools[selected].BDPool[db]
		if !ok {
			rp.rPools[selected].BDPool[db] = newPool(rp.rPools[selected].Address, rp.maxActiveConn, db)
			pool = rp.rPools[selected].BDPool[db]
		}
	}
	return pool.Get()
}
func (rp *RedisPool) GetWriteConnection(db int) redis.Conn {
	pool, ok := rp.wPool.BDPool[db]
	if !ok {
		rp.wPool.BDPool[db] = newPool(rp.wPool.Address, rp.maxActiveConn, db)
		pool = rp.wPool.BDPool[db]
	}
	return pool.Get()
}
