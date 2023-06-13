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

func FromKeyCacheRandomItemKey(n int, seed int64) Key {
	r := rand.New(rand.NewSource(seed)) // Use the passed in seed for random number generation

	items := KeysCache.Items()
	PlusKeys := []Key{}
	OtherKeys := []Key{}

	// Separate keys into PlusKeys and OtherKeys
	for _, item := range items {
		key, ok := item.Object.(Key)
		if ok {
			if key.ApiType == "openai-plus" {
				PlusKeys = append(PlusKeys, key)
			} else {
				OtherKeys = append(OtherKeys, key)
			}
		}
	}

	r.Shuffle(len(PlusKeys), func(i, j int) {
		PlusKeys[i], PlusKeys[j] = PlusKeys[j], PlusKeys[i]
	})

	r.Shuffle(len(OtherKeys), func(i, j int) {
		OtherKeys[i], OtherKeys[j] = OtherKeys[j], OtherKeys[i]
	})

	// If n is less than the number of PlusKeys, select a key from PlusKeys
	if n < len(PlusKeys) {
		selectedKey := PlusKeys[n]
		log.Printf("Selected Key: {Name: %s, ApiType: %s}\n", selectedKey.Name, selectedKey.ApiType)
		return selectedKey
	}

	// If n is greater than or equal to the number of PlusKeys, select a key from OtherKeys
	if len(OtherKeys) > 0 {
		selectedKey := OtherKeys[n%len(OtherKeys)]
		log.Printf("Selected Key: {Name: %s, ApiType: %s}\n", selectedKey.Name, selectedKey.ApiType)
		return selectedKey
	}

	log.Println("No keys available")
	return Key{} // Return an empty Key if there are no keys
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
