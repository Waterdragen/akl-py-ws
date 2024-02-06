"""
Set the path to the cmini websocket
example: ws://localhost:8000/<path>
"""
import re

from django.urls import re_path

from . import consumers

WEBSOCKET_URLPATTERNS = [
    re_path("^python/cmini(/)*$", consumers.CminiConsumer.as_asgi()),
    re_path("^python/a200(/)*$", consumers.A200Consumer.as_asgi()),
]
