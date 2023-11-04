package store

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/Sakurasan/to"
	"github.com/patrickmn/go-cache"
)

var (
	KeysCache *cache.Cache
	AuthCache *cache.Cache
)

func init() {
	KeysCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	AuthCache = cache.New(cache.NoExpiration, cache.NoExpiration)
}

func LoadKeysCache() {
	KeysCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	keys, err := GetAllKeys()
	if err != nil {
		log.Println(err)
		return
	}
	for idx, key := range keys {
		KeysCache.Set(to.String(idx), key, cache.NoExpiration)
	}
}

func FromKeyCacheRandomItemKey() Key {
	items := KeysCache.Items()
	if len(items) == 1 {
		return items[to.String(0)].Object.(Key)
	}
	idx := rand.Intn(len(items))
	item := items[to.String(idx)]
	return item.Object.(Key)
}

func SelectKeyCache(apitype string) (Key, error) {
	var keys []Key
	items := KeysCache.Items()
	for _, item := range items {
		if item.Object.(Key).ApiType == apitype {
			keys = append(keys, item.Object.(Key))
		}
	}
	if len(keys) == 0 {
		return Key{}, errors.New("No key found")
	} else if len(keys) == 1 {
		return keys[0], nil
	}
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(keys))
	return keys[idx], nil
}

func LoadAuthCache() {
	AuthCache = cache.New(cache.NoExpiration, cache.NoExpiration)
	users, err := GetAllUsers()
	if err != nil {
		log.Println(err)
		return
	}
	for _, user := range users {
		AuthCache.Set(user.Token, true, cache.NoExpiration)
	}
}

func IsExistAuthCache(auth string) bool {
	items := AuthCache.Items()
	_, ok := items[auth]
	return ok
}
