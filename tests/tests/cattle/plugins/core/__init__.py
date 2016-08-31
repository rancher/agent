import api_proxy
import event_handlers
import event_router
import marshaller
from tests.cattle.type_manager import POST_REQUEST_HANDLER, LIFECYCLE
from tests.cattle.type_manager import register_type, MARSHALLER, ROUTER

register_type(MARSHALLER, marshaller.Marshaller())
register_type(ROUTER, event_router.Router())
register_type(POST_REQUEST_HANDLER, event_handlers.PingHandler())
register_type(POST_REQUEST_HANDLER, event_handlers.ConfigUpdateHandler())
register_type(LIFECYCLE, api_proxy.ApiProxy())
