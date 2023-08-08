-- redis lua位移操作只支持32位，并且是当做有符号数来处理，因此自己封装获取函数
 local function rshift(n, b)
    return math.floor(n / 2^b)
end

-- 入参value限制最大uint32
local function low16bit(value)
    return bit.band(value,0xFFFF)
end 

-- 入参value限制最大int64
local function low32bit(value)
    return value%0x100000000
end 

local function to_hex(value)
    return string.format("%x",value) 
end 
--------------------------------------------------------------------------RoaringBitmap64 implementation--------------------------------------------------------------------------

local RoaringBitmap64 = {}
RoaringBitmap64.__index = RoaringBitmap64

local rb64_prefix = "RB64:" 
local rb32_prefix = "RB32:"
local bit_count_prefix = "RB:LEN:" 
local container_prefix = "C:"

local function get_rb64_key(key)
    return rb64_prefix .. key
end

local function get_rb32_key(key, hex_high32)
    return rb32_prefix .. key .. hex_high32
end 

local function get_bit_count_key(key)
    return bit_count_prefix .. key
end

local function get_container_key(key, hex_high16)
    return container_prefix .. key .. hex_high16
end

-- 构造函数
function RoaringBitmap64:new(name, value)
    local self = setmetatable({}, RoaringBitmap64)
    self.name = name 
    self.high32 = rshift(value, 32)
    self.low32 = low32bit(value)
    self.high16 = rshift(self.low32, 16)
    self.low16 = low16bit(self.low32)
    return self
end  

local function add(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    -- 记录该rb64下有哪些rb32
    redis.call("SADD", get_rb64_key(rb64.name), to_hex(rb64.high32))
    -- 记录该rb32下有哪些container
    redis.call("SADD", get_rb32_key(rb64.name, to_hex(rb64.high32)), to_hex(rb64.high16))
    local result = redis.call("SETBIT", get_container_key(rb64.name, to_hex(rb64.high16)), rb64.low16, 1)
    if result == 0 then
        redis.call("INCRBY", get_bit_count_key(rb64.name), 1) 
    end 
end 

local function delete(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    local result = redis.call("SETBIT", get_container_key(rb64.name, to_hex(rb64.high16)), rb64.low16, 0)
    if result == 1 then
        redis.call("INCRBY", get_bit_count_key(rb64.name), -1) 
    end 
end 

local function contains(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    return redis.call("GETBIT", get_container_key(rb64.name, to_hex(rb64.high16)), rb64.low16)
end 

local function len(key)
    local count = redis.call("GET", get_bit_count_key(key))
    if count then
        return tonumber(count)
    end 
    return 0
end 

local function is_empty(key)
    return len(key) == 0 and 1 or 0 
end 

local function clear(key)
    local keys = {get_bit_count_key(key),get_rb64_key(key)}
    for i, hex_high32 in pairs(redis.call("SMEMBERS", get_rb64_key(key))) do 
        table.insert(keys, get_rb32_key(key, hex_high32))
        for j, hex_high16 in pairs(redis.call("SMEMBERS", get_rb32_key(key, hex_high32))) do
            table.insert(keys, get_container_key(key, hex_high16))
        end 
    end 
    -- unpack有最大key数量限制,这里分批删除
    for i = 1, #keys, 7000 do
        redis.call("UNLINK", unpack(keys, i, math.min(i+6999, #keys))) 
    end 
end 

--------------------------------------------------------------------------redis RoaringBitmap64 usage--------------------------------------------------------------------------

local name = KEYS[1]
local op = ARGV[1] 

if op == 'Add' then
    add(name, ARGV[2]) 
elseif op == 'Remove' then 
    delete(name, ARGV[2]) 
elseif op == 'Contains' then 
    return contains(name, ARGV[2])
elseif op == 'Len' then  
    return len(name)
elseif op == 'IsEmpty' then
    return is_empty(name)
elseif op == 'Clear' then
    clear(name)
end