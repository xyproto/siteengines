package siteengine

type UserState struct {
	// see: http://redis.io/topics/data-types
	users       *RedisHashMap   // Hash map of users, with several different fields per user ("loggedin", "confirmed", "email" etc)
	usernames   *RedisSet       // A list of all usernames, for easy enumeration
	unconfirmed *RedisSet       // A list of unconfirmed usernames, for easy enumeration
	pool        *ConnectionPool // A connection pool for Redis
}

