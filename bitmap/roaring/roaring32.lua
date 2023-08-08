local function get_container_key(name, hex_hig16)
    return "CONTAINER:" .. name .. hex_hig16
end

local function get_bit_count_key(name)
    return "BITCOUNT:" .. name
end

local function get_container_list_key(name)
    return "CONTAINER:LIST:" .. name
end

local function to_hex(value)
    return string.format("%x",value) 
end 

local function add(name, value)
    local high16 = bit.rshift(value, 16)
    local low16 = bit.band(value, 0xFFFF)
    local hex_hig16 = to_hex(high16)

    redis.call("SADD",get_container_list_key(name), hex_hig16)
    local result = redis.call("SETBIT", get_container_key(name, hex_hig16), low16, 1)
    if result == 0 then
        redis.call("INCRBY", get_bit_count_key(name), 1) 
    end 
end 

local function delete(name, value)
    local high16 = bit.rshift(value, 16)
    local low16 = bit.band(value, 0xFFFF)
    local hex_hig16 = to_hex(high16)

    local result = redis.call("SETBIT", get_container_key(name, hex_hig16), low16, 0)
    if result == 1 then
        redis.call("INCRBY", get_bit_count_key(name), -1) 
    end 
end 

local function contains(name, value)
    local high16 = bit.rshift(value, 16)
    local low16 = bit.band(value, 0xFFFF)
    local hex_hig16 = to_hex(high16)

    return redis.call("GETBIT", get_container_key(name, hex_hig16), low16) == 1
end 

local function len(name)
    local count = redis.call("GET", get_bit_count_key(key))
    if count then
        return tonumber(count)
    end 
    return 0
end 

local function is_empty(name)
    return len(key) == 0 and 1 or 0 
end 

local function clear(name)
    local keys = {get_bit_count_key(name),get_container_list_key(name)}
    for i, hex_high16 in pairs(redis.call("SMEMBERS", get_container_list_key(name))) do
        table.insert(keys, get_container_key(name, hex_high16))
    end
    -- unpack有最大key数量限制,这里分批删除
    for i = 1, #keys, 7000 do
        redis.call("UNLINK", unpack(keys, i, math.min(i+6999, #keys))) 
    end 
end 

--------------------------------------------------------------------------redis RoaringBitmap32 usage--------------------------------------------------------------------------

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