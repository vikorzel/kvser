package.path = 'lua/?.lua;' .. package.path
require("strict")

local http_server = require("http.server")
local http_processing = require("modules.http_processing")
local config = require("modules.config")
local log = require("log")
local fiber = require("fiber")



function init_routes(server)
    server:route(
        {
            method = "GET",
            path   = "/kv/.*" 
        },
        http_processing.handler
    )
    server:route(
        {
            method = "POST",
            path   = "/kv"
        },
        http_processing.handler
    )
    server:route(
        {
            method = "PUT",
            path   = "/kv/.*"
        },
        http_processing.handler
    )
    server:route(
        {
            method = "DELETE",
            path   = "/kv/.*"
        },
        http_processing.handler
    )
    
end

log.info("APP: Config parsing")
local cfg = config.parse_config("/opt/tarantool/config/serverconfig.json")
log.info("APP: Config parsed")

log.info("APP: Box init")
box.cfg{
    log="log/applog.log",
    log_level=7
}
log.info("APP: Box inited")


log.info("APP: Server instance create")
local httpd = http_server.new( cfg.address, cfg.port, {
    log_requests = true,
    log_errors = true
})
log.info("APP: Server instance created")

require("modules.db_controller").init_db()

log.info("APP: Start QPS shaper")
fiber.create(http_processing.start_shaper,cfg.qps_limit)
log.info("APP: QPS shaper started")

log.info("APP: Init routes")
init_routes(httpd)
log.info("APP: Routes was inited")

log.info("APP: Start HTTP server")
httpd:start()
