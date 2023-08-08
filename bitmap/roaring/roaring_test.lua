-- 该版本使用本地操作替换了roaring.lua中的对redis的操作，用于做性能测试。
local bit = require("bit32")
local cmsgpack = require("msgpack")

-- redis lua位移操作只支持32位，并且是当做有符号数来处理，因此自己封装获取函数
local function lshift(n, b)
    return n * 2^b
end
 
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

-- bitmap设值,并返回是否是新设进去的值
local function set_bit(bitmap, value)
    local byte_index = math.floor(value / 8) + 1
    local bit_index = value % 8
    local byte_before = bitmap[byte_index] or 0
    bitmap[byte_index] = bit.bor(byte_before, bit.lshift(1, bit_index))
    return byte_before ~= bitmap[byte_index]
end  

-- bitmap清除值,并返回是否是真正清除了
local function clear_bit(bitmap, value)
    local byte_index = math.floor(value / 8) + 1
    local bit_index = value % 8
    local byte_before = bitmap[byte_index] or 0
    bitmap[byte_index] = bit.band(byte_before, bit.bnot(bit.lshift(1, bit_index)))
    return byte_before ~= bitmap[byte_index]
end

-- bitmap判断值存在
local function get_bit(bitmap, value)
    local byte_index = math.floor(value / 8) + 1
    local bit_index = value % 8
    return bit.band(bitmap[byte_index] or 0, bit.lshift(1, bit_index)) ~= 0
end

-- 二分查找：返回元素value应该在的位置
local function binarySearch(array, value)
    local left = 1
    local right = #array
    while left <= right do
        local mid = math.floor((left + right) / 2)
        if array[mid] == value then
            return mid 
        elseif array[mid] < value then
            left = mid + 1
        else
            right = mid - 1
        end
    end
    return left 
end

-- 插入值到数组对应位置，并返回是否是新插入
local function insert(array, loc, value)
    local exist = (array[loc] == value)
    if exist then
        return false 
    else
        table.insert(array, loc, value)
        return true 
    end  
end

-- 删除数组对应位置的值，并返回是否执行了删除
local function remove(array, loc, value)
    local exist = (array[loc] == value)
    if exist then
        table.remove(array, loc)
        return true 
    else
        return false 
    end  
end

-- 容器类型 支持roaring中的array_container和bitmap_container
local Container = {}
Container.__index = Container

-- 容器类型构造函数
function Container:new()
    local self = setmetatable({}, Container)   
    self.array_container = {}
    self.bitmap_container = {}
    self.is_array = true
    return self
end

-- 容器方法：设置元素 元素数量小于4096时使用array_container，否则使用bitmap_container
function Container:add(low_16bit)
    if self.is_array then
        -- 根据roaring原理，array中每个元素占2字节，当元素个数为4096时刚占用8K内存
        -- 而bit_mapcontainer始终占用2^16=8K，所以此时将array_container转化为bitmap_container更节省内存
        if #self.array_container >= 4096 then
            self:array_container_to_bitmap_container()
            return set_bit(self.bitmap_container, low_16bit)
        else
            local loc = binarySearch(self.array_container, low_16bit)
            return insert(self.array_container, loc, low_16bit)
        end
    else
        return set_bit(self.bitmap_container, low_16bit)
    end
end  

-- 容器方法：删除元素 删除元素时，不会进行容器类型转换，即不会从bitmap_container转化为array_container
-- 保证不会出现bitmap_container与array_container频繁互转导致性能损耗
function Container:delete(low_16bit)
    if self.is_array then
        local loc = binarySearch(self.array_container, low_16bit)
        return remove(self.array_container, loc, low_16bit)
    else
        return clear_bit(self.bitmap_container,low_16bit)
    end
end

-- 容器方法：判断元素是否存在
function Container:contains(low_16bit)
    if self.is_array then
        local loc = binarySearch(self.array_container, low_16bit)
        return self.array_container[loc] == low_16bit
    else
        return get_bit(self.bitmap_container, low_16bit)
    end
end

-- 将array类型容器转化为bitmap类型容器
function Container:array_container_to_bitmap_container()
    for i,v in ipairs(self.array_container) do
        set_bit(self.bitmap_container,v)
    end
    self.array_container = nil
    self.is_array = false
end 

-- 序列化
function Container:serialize()
    return cmsgpack.pack(self)
end

-- 反序列化
function Container:deserialize(data)
    local deserialize_data = cmsgpack.unpack(data)
    self.is_array = deserialize_data.is_array
    self.array_container = deserialize_data.array_container
    self.bitmap_container = deserialize_data.bitmap_container
end 

--------------------------------------------------------------------------RoaringBitmap64 implementation--------------------------------------------------------------------------

local RoaringBitmap64 = {}
RoaringBitmap64.__index = RoaringBitmap64

local rb64_prefix = "RB64:" 
local rb32_prefix = "RB32:"
local bit_count_prefix = "RB:LEN:" 

local count = 0 
local rb32_map = {}
local rb32_name_set_map = {}

local function get_rb64_key(key)
    return rb64_prefix .. key
end

local function get_bit_count_key(key)
    return bit_count_prefix .. key
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

function RoaringBitmap64:get_rb32_key()
    return rb32_prefix .. self.name .. string.format("%x",self.high32)
end 

function RoaringBitmap64:get_high16_field()
    return string.format("%x",self.high16)
end 

function RoaringBitmap64:get_high32_field()
    return string.format("%x",self.high32)
end  

local function add(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    -- 获取对应rb32
    local rb32 = rb32_map[rb64:get_rb32_key()]
    if rb32 == nil then 
        rb32 = {}
    end

    -- 获取对应container
    local  serialized_container = rb32[rb64:get_high16_field()]
    local container = Container:new()
    if serialized_container then 
        container:deserialize(serialized_container)
    end 

    -- 执行set
    local added = container:add(rb64.low16)

    -- 设置rb32对应container
    rb32[rb64:get_high16_field()] = container:serialize()
    -- 设置rb32
    rb32_map[rb64:get_rb32_key()] = rb32

    -- 记录该rb64下有哪些rb32 用于清理
    local rb32_name_set = rb32_name_set_map[get_rb64_key(rb64.name)]
    if rb32_name_set == nil then
        rb32_name_set = {}
    end 
    rb32_name_set[ rb64:get_rb32_key()] = true 
    rb32_name_set_map = rb32_name_set

    -- 成功add一个数,计数+1
    if added then
        count = count + 1
    end 
end 

local function delete(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    -- 获取对应rb32
    local rb32 = rb32_map[rb64:get_rb32_key()]
    if rb32 == nil then 
        rb32 = {}
    end

    -- 获取对应container
    local  serialized_container = rb32[rb64:get_high16_field()]
    local container = Container:new()
    if serialized_container then 
        container:deserialize(serialized_container)
    end

    -- 执行删除
    local deleted = container:delete(rb64.low16)

    -- 设置rb32对应container
    rb32[rb64:get_high16_field()] = container:serialize()
    -- 设置rb32
    rb32_map[rb64:get_rb32_key()] = rb32
    if deleted then
        count = count - 1
    end 
end 

local function contains(key, value)
    local rb64 = RoaringBitmap64:new(key, value)

    -- 获取对应rb32
    local rb32 = rb32_map[rb64:get_rb32_key()]
    if rb32 == nil then 
        rb32 = {}
    end

    -- 获取对应container
    local  serialized_container = rb32[rb64:get_high16_field()]
    local container = Container:new()
    if serialized_container then 
        container:deserialize(serialized_container)
    else 
        return false
    end 
    return container:contains(rb64.low16)
end 

local function len(key)
    return count 
end 

local function is_empty(key)
    return count == 0
end 

local function clear(key)
    if #rb32_name_set_map == 0 then
        return  
    end 
    for i, v in pairs(rb32_name_set_map) do
        rb32_map[i] = nil 
    end
    rb32_name_set_map = {}
    count = 0 
end 

function test(num, random)
    print(string.format("test %d elements in random:%s",num, tostring(random)))
    local key = "test"
    local rands = {}
    for i = 1, num do
        if random then 
            rands[i] = math.random(0x7FFFFFFFFFFFFFFF) 
        else 
            rands[i] = i
        end 
    end 

    local now = os.clock()
    for i = 1, num do 
        add(key,rands[i])
    end
    local elapsed =  os.clock() - now
    print("add cost:" .. elapsed*1000 .. "ms")

    local now = os.clock()
    for i = 1, num do 
        assert(contains(key,rands[i]))
    end
    local elapsed =  os.clock() - now
    print("contains cost:" .. elapsed*1000 .. "ms")

    local now = os.clock()
    len(key)
    local elapsed =  os.clock() - now
    print("len cost:" .. elapsed*1000 .. "ms")

    local now = os.clock()
    for i = 1, num do 
        delete(key,rands[i])
    end
    local elapsed =  os.clock() - now
    print("remove cost:" .. elapsed*1000 .. "ms")
    print("")
end

test(50000, true)
test(100000, true)
test(200000, true)
test(500000, true)
test(700000, true)
test(900000, true)
test(1000000, true)