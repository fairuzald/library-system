package constants

import "time"

// Cache keys and TTL
const (
	CacheKeyBook       = "book:"
	CacheKeyCategory   = "category:"
	CacheKeyUser       = "user:"
	CacheKeyBooks      = "books:"
	CacheKeyCategories = "categories:"
	CacheKeyUsers      = "users:"

	CacheDefaultTTL = 15 * time.Minute
	CacheLongTTL    = 1 * time.Hour
	CacheShortTTL   = 5 * time.Minute
)
