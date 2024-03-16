"""
Defines what the server side should do after connection
"""
import traceback
from abc import abstractmethod, ABCMeta

from channels.generic.websocket import AsyncWebsocketConsumer

from .cmini.core.resource import CminiData
from .cmini.main import get_cmini_response
from .a200.src.core.resource import A200Data
from .a200.src.main import A200

from typing import TypeVar, Generic

T = TypeVar('T')

class BaseConsumer(AsyncWebsocketConsumer, Generic[T], metaclass=ABCMeta):
    sessions: dict[int, T]

    @property
    @abstractmethod
    def sessions(self):
        ...

    @abstractmethod
    def default_data(self) -> T:
        ...

    async def connect(self):
        await self.accept()
        session_id = id(self)
        self.sessions[session_id] = self.default_data()

    async def disconnect(self, code):
        self.sessions.pop(id(self), None)

    def get_session(self) -> T:
        return self.sessions[id(self)]


class CminiConsumer(BaseConsumer):
    sessions: dict[int, dict[str, str]] = {}

    def default_data(self) -> dict[str, str]:
        return {"corpus": "shai"}

    async def receive(self, text_data: str = None, bytes_data=None):
        try:
            # TODO: pass instance/method/dict into cmini
            session = self.get_session()
            user_data = CminiData(text_data, session)
            cmini_response = get_cmini_response(user_data)
            assert cmini_response != ""
        except Exception as e:
            traceback.print_exc()
            return
        if cmini_response is None:
            return

        await self.send(text_data=cmini_response)


class A200Consumer(BaseConsumer):
    sessions: dict[int, A200Data] = {}

    def default_data(self) -> A200Data:
        return A200Data(cache={}, config=None)

    async def receive(self, text_data=None, bytes_data=None):
        commands = str.split(text_data, " ")

        try:
            a200_console_log = A200(self).main(commands)
            await self.send(text_data=a200_console_log)
        except Exception as e:
            traceback.print_exc()
            traceback_string: str = traceback.format_exc()
            await self.send(text_data=traceback_string)

    def get_cache(self, cachefile: str) -> dict | None:
        session: A200Data = self.get_session()
        cache = session.cache.get(cachefile, None)
        return cache

    def get_config(self) -> dict | None:
        session: A200Data = self.get_session()
        return session.config

    def update_cache(self, cache_dict: dict, cachefile: str):
        session: A200Data = self.get_session()
        session.cache[cachefile] = cache_dict

    def clear_cache(self):
        session: A200Data = self.get_session()
        session.cache = {}

    def update_config(self, config_dict: dict):
        session: A200Data = self.get_session()
        session.config = config_dict
