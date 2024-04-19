package queue

import "github.com/go-redis/redis/v8"

/*
 * @Time   : 2021/1/19 下午20:00
 * @Email  : jingjing.yang@tvb.com
 */

// luaScripts redis lua
type luaScripts struct{}

// 定义lua script
var (
	size = redis.NewScript(`
return redis.call('llen', KEYS[1]) + redis.call('zcard', KEYS[2]) + redis.call('zcard', KEYS[3])
`)
	pop = redis.NewScript(`
-- Pop the first job off of the queue...
local job = redis.call('lpop', KEYS[1])
local reserved = false
local timeoutAt = 0

if(job ~= false) then
	-- Increment the attempt count and place job on the reserved queue...
	reserved = cjson.decode(job)
	-- if first pop time less then 0 , set now int unix time
	if reserved['PopTime'] <= 0 then
		reserved['PopTime'] = tonumber(ARGV[1])
	end
	-- calc next attempts time
	timeoutAt = tonumber(ARGV[1]) + tonumber(reserved['Timeout'])
	-- set reserved val
	reserved['Attempts'] = reserved['Attempts'] + 1
	reserved['TimeoutAt'] = timeoutAt
	-- encode to string
	reserved = cjson.encode(reserved)
	-- set next attempt time as
	redis.call('zadd', KEYS[2], timeoutAt, reserved)
end

return {job, reserved}
`)
	release = redis.NewScript(`
-- Remove the job from the current queue...
redis.call('zrem', KEYS[2], ARGV[1])

-- Add the job onto the "delayed" queue...
redis.call('zadd', KEYS[1], ARGV[2], ARGV[1])

return true
`)
	migrate = redis.NewScript(`
-- Get all of the jobs with an expired "score"...
local val = redis.call('zrangebyscore', KEYS[1], '-inf', ARGV[1])

-- If we have values in the array, we will remove them from the first queue
-- and add them onto the destination queue in chunks of 100, which moves
-- all of the appropriate jobs onto the destination queue very safely.
if(next(val) ~= nil) then
    redis.call('zremrangebyrank', KEYS[1], 0, #val - 1)

    for i = 1, #val, 100 do
        redis.call('rpush', KEYS[2], unpack(val, i, math.min(i+99, #val)))
    end
end

return val
`)
)

// Size
/**
 * Get the Lua script for computing the size of queue.
 *
 * KEYS[1] - The name of the primary queue
 * KEYS[2] - The name of the "delayed" queue
 * KEYS[3] - The name of the "reserved" queue
 *
 * @return string
 */
func (lua *luaScripts) Size() *redis.Script {
	return size
}

// Pop
/**
 * Get the Lua script for popping the next job off of the queue.
 *
 * KEYS[1] - The queue to pop jobs from, for example: queues:foo
 * KEYS[2] - The queue to place reserved jobs on, for example: queues:foo:reserved
 * ARGV[1] - The Now unix time
 *
 * @return string
 */
func (lua *luaScripts) Pop() *redis.Script {
	return pop
}

// Release
/**
 * Get the Lua script for releasing reserved jobs.
 *
 * KEYS[1] - The "delayed" queue we release jobs onto, for example: queues:foo:delayed
 * KEYS[2] - The queue the jobs are currently on, for example: queues:foo:reserved
 * ARGV[1] - The raw payload of the job to add to the "delayed" queue
 * ARGV[2] - The UNIX timestamp at which the job should become available
 *
 * @return string
 */
func (lua *luaScripts) Release() *redis.Script {
	return release
}

// MigrateExpiredJobs
/**
 * Get the Lua script to migrate expired jobs back onto the queue.
 *
 * KEYS[1] - The queue we are removing jobs from, for example: queues:foo:reserved
 * KEYS[2] - The queue we are moving jobs to, for example: queues:foo
 * ARGV[1] - The current UNIX timestamp
 *
 * @return string
 */
func (lua *luaScripts) MigrateExpiredJobs() *redis.Script {
	return migrate
}
