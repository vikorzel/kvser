local exports ={}

local db_controller = require("modules.db_controller")
local constants = require("constants")
local utils = require("modules.utils")
local log = require("log")
local http_server = require("http.server")
local json = require("json")
local fiber = require("fiber")


local allowed_slots = 0
local http_codes = {
    ok              =200,
    created         =201,
    bad_request     =400,
    not_found       =404,
    conflict        =409,
    internal_err    =500,
    
}

exports.start_shaper = function(limit) 
    if not limit or limit == 0 then
        allowed_slots = -1
        return
    end

    allowed_slots = limit
    
    while true do
        if (allowed_slots < limit ) then 
            allowed_slots = allowed_slots + 1
        end
        fiber.sleep(1/limit)
    end
end

function is_allowed_new_request()  
    if allowed_slots > 0 then
        log.debug("HTTP: Request passed shaper")
        allowed_slots = allowed_slots - 1
        return true
    end
    
    if allowed_slots == -1 then
        log.debug("HTTP: Request passed shaper. There is no limit")
        return true
    end
    log.debug("HTTP: Request rejected by QPS limit")
    return false
end

function _get_id_from_uri(req)
    local uri = req.path
    return uri:len() > 4 and uri:sub(5) or nil
end

function _get_response_by_db_err_code(code,ok_code,id)
    local response = {status = http_codes.internal_err, body = constants.messages.UNKNOWN_ERROR_CODE .. string.format("[%s]", code)}
    log.debug("HTTP: Lookup response for errorcode", code or "<nil>")
    if code == nil then
        response = {status = ok_code, body = id}
    elseif code ==  db_controller.errors.KEY_EXISTS then
        response = {status = http_codes.conflict, body = constants.messages.KEY_EXISTS_IN_DB .. string.format( "[%s]",id )}
    elseif code == db_controller.errors.BROKEN_JSON then
        response = {status=http_codes.bad_request, body = constants.messages.BROKEN_JSON}
    elseif code == db_controller.errors.KEY_NOT_EXISTS then
        response = {status=http_codes.not_found, body = constants.messages.KEY_NOT_EXISTS_IN_DB .. string.format("[%s]", id)}
    end
    log.debug("HTTP: Response code will be", response.status)
    return response
end

function onPOST(env)
    local data = env:read_cached()
    log.debug("HTTP: POST received:", data)
    local valid, json  = pcall(json.decode, data)
    if not valid or not json.key or not json.value then
        local reason = ""
        if     not valid      then reason="Cannot parse JSON"    
        elseif not json.key   then reason="Key is not defined"
        elseif not json.value then reason="Value is not defined" end
        log.warn("HTTP: POST cannot parse request:", reason)
        return  {status=http_codes.bad_request, body = reason}
    end
    log.debug("HTTP: POST parsed json.", data)
    local err = db_controller.append(tostring(json.key), json.value)
    return _get_response_by_db_err_code(err, http_codes.created, json.key)
end

function onGET(env)
    local id = _get_id_from_uri(env)
    local body, err = db_controller.get_json(id)
    if err == nil then 
        log.debug("HTTP: GET finished successfully. Body:",body)
        return {status = http_codes.ok, body = body}
    else
        log.warn("HTTP: GET error")
        return _get_response_by_db_err_code(err,http_codes.ok,id)
    end
end

function onPUT(env)
    local id = _get_id_from_uri(env)
    local err = db_controller.update(id, env:read_cached())
    return _get_response_by_db_err_code(err,http_codes.ok,id)
end

function onDELETE(env)
    local id = _get_id_from_uri(env)
    local err = db_controller.delete(id)
    return _get_response_by_db_err_code(err,http_codes.ok,id)
end

exports.handler = function ( env )
    if not is_allowed_new_request() then return constants.REQUEST_LIMIT_RESPONSE end
    log.warn(string.format("HTTP: new %s request", env.method))
    if(env.method == "GET") then
        return onGET(env)
    elseif(env.method == "PUT") then
        return onPUT(env)
    elseif(env.method == "DELETE") then
        return onDELETE(env)
    elseif(env.method == "POST") then
        return onPOST(env)
    end
   
    return constants.UNKNOUWN_URI_RESPONSE
end

return exports