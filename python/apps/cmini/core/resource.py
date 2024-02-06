import glob
import os

from dataclasses import dataclass
from typing import Literal


@dataclass
class CminiData:
    message: str
    user_cache: dict[str, str]

    def get_corpus(self) -> str:
        return self.user_cache["corpus"]

    def set_corpus(self, name: str):
        self.user_cache["corpus"] = name


class AbsPath:
    def __init__(self, dunder_file: str):
        self.py_file_path = dunder_file

    def get_real_dir(self, pathname: str):
        real_path = os.path.realpath(self.py_file_path)
        return os.path.join(os.path.dirname(real_path), pathname)

    def open(self, pathname: str, mode: Literal['r', 'w']):
        return open(self.get_real_dir(pathname),
                    mode=mode,
                    encoding="utf-8")

    def glob(self, pathname: str):
        return glob.glob(self.get_real_dir(pathname))

    @staticmethod
    def get_basename(fullpath: str) -> str:
        return os.path.splitext(os.path.basename(fullpath))[0]

