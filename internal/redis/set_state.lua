redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[4])
redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[4])
redis.call("SET", KEYS[3], ARGV[3])

return "OK"