from ..core.resource import CminiData
from ..util import layout, memory, parser

RESTRICTED = False

def exec(user: CminiData):
    name = parser.get_arg(user)
    ll = memory.find(name.lower())

    if not ll:
        return f'Error: could not find layout `{name}`'

    return layout.to_string(ll, user)

def use():
    return 'view [name]'

def desc():
    return 'see the stats of a layout'
