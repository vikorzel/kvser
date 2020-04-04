local exports = {}
function parse_config( path )
    local json = require("json")
    local file = io.open(path, 'r')
    if not file then return nil end
    local raw_content = file:read "*all"
    local valid, result = pcall(json.decode, raw_content)
    if not valid then return nil end
    return result
end

exports.parse_config = parse_config
return exports