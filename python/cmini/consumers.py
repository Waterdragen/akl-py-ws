"""
Defines what the server side should do after connection
"""
from abc import abstractmethod, ABCMeta

from channels.generic.websocket import AsyncWebsocketConsumer
from .src.main import get_cmini_response
import traceback

from typing import Any

class BaseConsumer(AsyncWebsocketConsumer, metaclass=ABCMeta):
    sessions: dict[int, dict] = {}

    async def connect(self):
        await self.accept()
        session_id = hash(self)
        self.sessions[session_id] = self.default_data()

    @abstractmethod
    def default_data(self) -> dict:
        ...

    async def disconnect(self, code):
        self.sessions.pop(hash(self), None)


class CminiConsumer(BaseConsumer):
    def default_data(self) -> dict[str, str]:
        return {"corpus": "shai"}

    async def receive(self, text_data: str = None, bytes_data=None):
        try:
            cmini_response = get_cmini_response(text_data)
        except Exception as e:
            traceback.print_exc()
            return
        if cmini_response is None:
            return
        await self.send(text_data=get_cmini_response(text_data))


class A200Consumer(BaseConsumer):
    def default_data(self) -> dict[str, int]:
        return {"message_count": 0}

    async def receive(self, text_data=None, bytes_data=None):
        session_id = hash(self)

        # Increment the message count for the session
        session = self.sessions[session_id]
        session["message_count"] += 1

        await self.send(text_data=str(session["message_count"]))

