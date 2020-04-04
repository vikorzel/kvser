local json = require("json")
local log = require("log")


local keys = {
    id = 1,
    json = 2
}

local exports = {}
exports.errors = {
    KEY_EXISTS = 1,
    BROKEN_JSON = 2,
    KEY_NOT_EXISTS = 3
}



function init_db()
    log.debug("DB: init started")
    local space = box.schema.space.create("jstor", {
        if_not_exists = true
    })
    space:format({
        {name="id", type = "string"},
        {name="json", type = "map"}
    })
    space:create_index('primary', {type="HASH", parts={"id"}, if_not_exists = true})
    log.debug("DB init finished")
end

function append(id, json)
    local err 
    log.debug("DB: append started")
    local ok = pcall(function() box.space.jstor:insert{id, json} end)
    if not ok then
        log.warn(string.format("DB: append failed. Key %s already exists.", id))
        return exports.errors.KEY_EXISTS
    end
    log.debug("DB: append finished successfully:", id, json)
    return nil
end

function get_json(id)
    log.debug("DB: get started")
    local ret = box.space.jstor:select(id)
    if ret[1] == nil then
        log.warn(string.format("DB: get faled. Key %s does not exists", id))
        return nil, exports.errors.KEY_NOT_EXISTS
    end
    log.debug("DB: get finished succesfully:", id)
    return json.encode(ret[1][keys.json]), nil
end

function update(id, json_str)
    log.debug("DB: update started")
    local err
    local valid, deserialized = pcall(json.decode, json_str)
    
    if not valid then 
        log.warn("DB: update failed. Json is broken: ", json_str)
        return exports.errors.BROKEN_JSON
    end
    local ret = box.space.jstor:update(id, {{'=', keys.json, deserialized}})
    if ret then 
        log.debug("DB: update finished successfully:", id, json_str)
        return nil
    else
        log.warn(string.format("DB: update failed. Key %s does not exists",id))
        return  exports.errors.KEY_NOT_EXISTS
    end
end

function delete(id)
    log.debug("DB: delete started")
    local ret = box.space.jstor:delete(id)
    if not ret then
        log.warn(string.format("DB: delete failed. Key %s does not exists", id))
        return exports.errors.KEY_NOT_EXISTS
    end
    log.debug("DB: delete finished successfully:", id)
    return nil
end

exports.delete = delete
exports.update = update
exports.get_json = get_json
exports.append = append
exports.init_db = init_db

return exports