from ..core.resource import CminiData
from ..util import layout, memory, parser
from ..core.keyboard import Layout

RESTRICTED = False

def exec(user: CminiData):
    name = parser.get_arg(user)
    ll: Layout = memory.find(name.lower())

    if not ll:
        return f'Error: couldn\'t find any layout named `{name}`'

    modify(ll)
    ll.name += ' (non angle modded)'

    return layout.to_string(ll, user)

def modify(ll: Layout) -> None:
    if ll.board == 'angle':
        ll.board = 'ortho'
        for key in ll.keys.values():
            if key.row != 2:
                continue
            col = key.col
            if col >= 5:
                continue
            if col == 4:
                key.col = 0
                key.finger = 'LP'
            else:
                key.col += 1


def use():
    return 'unangle [layout_name]'

def desc():
    return 'view the non angle modded version of a layout'
