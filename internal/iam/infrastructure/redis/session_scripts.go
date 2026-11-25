package redis

import goredis "github.com/redis/go-redis/v9"

var rotateRefreshScript = goredis.NewScript(`
local usedUser = redis.call("GET", KEYS[4])
if usedUser then
	return { "REUSED", usedUser }
end
local raw = redis.call("GETDEL", KEYS[1])
if not raw then
	return nil
end
local data = cjson.decode(raw)
local sessionTTL = tonumber(ARGV[1])
local refreshTTL = tonumber(ARGV[2])
data.refresh_hash = ARGV[5]
data.expires_at = ARGV[7]
redis.call("SET", KEYS[2], cjson.encode(data), "EX", refreshTTL)
data.expires_at = ARGV[6]
redis.call("SET", KEYS[3], cjson.encode(data), "EX", sessionTTL)
redis.call("SET", KEYS[4], data.user_id, "EX", refreshTTL)
local userKey = "iam:user_sessions:" .. data.user_id
redis.call("SADD", userKey, ARGV[3], ARGV[4])
redis.call("EXPIRE", userKey, refreshTTL)
return { "OK", data.user_id }
`)
