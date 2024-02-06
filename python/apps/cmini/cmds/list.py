from ..util import authors, parser, memory
from ..core.resource import CminiData, AbsPath

def exec(user: CminiData):
    name = parser.get_arg(user)
    
    if name:
        id = authors.get_id(name)
        name = authors.get_name(id)
    else:
        return f"```\n" \
               f"{use()}\n" \
               f"    {desc()}\n" \
               f"```"

    if not id:
        return f'Error: user `{name}` does not exist'

    lines = [f'{name}\'s layouts:', '```']

    layouts = []
    for file in AbsPath(__file__).glob('../layouts/*.json'):
        ll = memory.parse_file(file)

        if ll.user == id:
            layouts.append(ll.name)

    lines += list(sorted(layouts))
    lines.append('```')

    return '\n'.join(lines)


def use():
    return 'list [username]'

def desc():
    return 'see a list of a user\'s layouts'
