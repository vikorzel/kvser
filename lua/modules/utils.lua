exports = {}
function json_valid_and_decode(str)
    local json = require("json")
    local valid, body =  pcall( function() json.decode(req.body) end)
    return valid, body
end

exports.decode_if_possible = json_valid_and_decode
return exports