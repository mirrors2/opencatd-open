package store

import (
	"log"
	"math/rand"

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
