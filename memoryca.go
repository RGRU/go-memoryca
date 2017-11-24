package memoryca

// TODO поиск по ключу и значению (регулярка), (второй парамерт countItem)
// Сортировка и вывод множественных значений (второй парамерт countItem)
// Инкримент, дикримент
// Rebase перенос из одной БД в другую
// Кол-во элементов в DB GetCount
// Невозможность регистрации с двумя одинаковыми БД

import (
	"errors"
	"sync"

	"time"
)

// Cache struct cache
type Cache struct {
	db                DataBase
	defaultExpiration time.Duration
	mu                sync.RWMutex
	availability      func(string, interface{})
	cleanupInterval   time.Duration
}

// DataBase struct
type DataBase struct {
	items map[string]Item
}

// Item struct cache item
type Item struct {
	Value      interface{}
	Expiration int64
	Created    time.Time
	Duration   time.Duration
}

// New initializing a new memory cache
func New(database string, defaultExpiration, cleanupInterval time.Duration) *Cache {

	db := make(map[string]DataBase)

	// database
	db[database] = DataBase{
		items: make(map[string]Item),
	}

	// cache item
	cache := Cache{
		db:                db[database],
		defaultExpiration: defaultExpiration,
		cleanupInterval:   cleanupInterval,
	}

	return &cache
}

// Set setting a cache by key
func (c *Cache) Set(key string, value interface{}, duration time.Duration) error {

	var expiration int64

	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	c.db.items[key] = Item{
		Value:      value,
		Expiration: expiration,
		Created:    time.Now(),
		Duration:   duration,
	}

	return nil

}

// Get getting a cache by key
func (c *Cache) Get(key string) (interface{}, bool) {

	c.mu.RLock()

	item, found := c.db.items[key]

	// cache not found
	if !found {
		c.mu.RUnlock()
		return nil, false
	}

	if item.Expiration > 0 {

		// cache expired
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, false
		}

	}

	c.mu.RUnlock()

	return item.Value, true
}

// GetItem getting item cache
// Second parameter returns false if cache not found or expired
func (c *Cache) GetItem(key string) (*Item, bool) {

	c.mu.RLock()

	item, found := c.db.items[key]

	// cache not found
	if !found {
		c.mu.RUnlock()
		return nil, false
	}

	if item.Expiration > 0 {

		// cache expired
		if time.Now().UnixNano() > item.Expiration {
			c.mu.RUnlock()
			return nil, false
		}

	}

	c.mu.RUnlock()

	return &item, true
}

// GetCount return count items in database
func (c *Cache) GetCount() int {

	return len(c.db.items)

}

// Delete cache by key
func (c *Cache) Delete(key string) error {

	c.mu.Lock()

	defer c.mu.Unlock()

	if _, found := c.db.items[key]; !found {
		return errors.New("Key not exist")
	}

	delete(c.db.items, key)

	return nil
}

// Exists check cache exist
func (c *Cache) Exists(key string) bool {

	c.mu.RLock()

	defer c.mu.RUnlock()

	if value, found := c.db.items[key]; found {
		return !value.Expire()
	}

	return false
}

// Expire check cache expire
// Return true if cache expired
func (i *Item) Expire() bool {

	if i == nil {
		return false
	}

	return time.Now().UnixNano() > i.Expiration
}

// FlushAll delete all keys in database
func (c *Cache) FlushAll() error {

	c.mu.Lock()

	defer c.mu.Unlock()

	// c.db.items = make(map[string]Item)

	for k := range c.db.items {
		delete(c.db.items, k)
	}

	return nil
}

// Rename rename key to newkey
// When renaming, the value with the key is deleted
// Return false in key eq newkey
// Return false if newkey is exist
func (c *Cache) Rename(key string, newKey string) error {

	if key == newKey {
		return errors.New("A key with this name already exists")
	}

	_, foundNewKey := c.db.items[newKey]

	if foundNewKey {
		return errors.New("Can not rename key")
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	value, found := c.db.items[key]

	if !found {
		return errors.New("Can not rename key")
	}

	c.db.items[newKey] = value

	delete(c.db.items, key)

	return nil
}

// Copy copying a value from key to a newkey
// Return error if key eq newkey
// Return error if key is empty
// Return error if newkey is exist
func (c *Cache) Copy(key string, newKey string) error {

	if key == newKey {
		return errors.New("The name of the keys can not be the same")
	}

	_, foundNewKey := c.db.items[newKey]

	if foundNewKey {
		return errors.New("There is already a key with that name")
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	value, found := c.db.items[key]

	if !found {
		return errors.New("Key not exist")
	}

	c.db.items[newKey] = value

	return nil
}

// StartGC start Garbage Collection
func (c *Cache) StartGC() error {

	go c.GC()

	return nil

}

// GC Garbage Collection
func (c *Cache) GC() {

	if c.cleanupInterval < 1 {

		return
	}

	for {

		<-time.After(c.cleanupInterval)

		if c.db.items == nil {
			return
		}

		// fmt.Println(c.expiredKeys())

		if keys := c.expiredKeys(); len(keys) != 0 {
			c.clearItems(keys)

		}

	}

}

// expiredKeys returns key list which are expired.
func (c *Cache) expiredKeys() (keys []string) {

	c.mu.RLock()

	defer c.mu.RUnlock()

	for key, i := range c.db.items {
		if i.Expire() {
			keys = append(keys, key)
		}
	}

	return
}

// clearItems removes all the items which key in keys.
func (c *Cache) clearItems(keys []string) {

	c.mu.Lock()

	defer c.mu.Unlock()

	for _, key := range keys {
		delete(c.db.items, key)
	}
}

// Get getting a cache by key without expire check
func (c *Cache) getWithOutExpire(key string) (interface{}, bool) {

	c.mu.RLock()

	item, found := c.db.items[key]

	// cache not found
	if !found {
		c.mu.RUnlock()
		return nil, false
	}

	c.mu.RUnlock()

	return item.Value, true
}

// // benchmarkGet benchmark set cahce
// func (c *Cache) benchmarkSet(b *testing.B) {
//
// 	for n := 0; n < b.N; n++ {
//
// 		c.Set("testKey:"+string(b.N), "testValue"+string(b.N), 1*time.Minute)
//
// 	}
//
// }
//
// // benchmarkGet benchmark get cahce
// func (c *Cache) benchmarkGet(b *testing.B) {
//
// 	for n := 0; n < b.N; n++ {
//
// 		c.Get("testKey:" + string(b.N))
//
// 	}
//
// }