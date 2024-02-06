import random

from ..util import layout, memory
from ..core.resource import CminiData, AbsPath

RESTRICTED = False

def exec(user: CminiData):
    files = AbsPath(__file__).glob('../layouts/*.json')
    file = random.choice(files)

    ll = memory.parse_file(file)

    return layout.to_string(ll, user)
