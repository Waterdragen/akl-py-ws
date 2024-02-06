from collections import Counter

from ..util import parser, memory
from ..core.resource import CminiData, AbsPath

def exec(user: CminiData):
    args = parser.get_args(user)

    if not args:
        return 'Error: please provide a letter.'

    arg = args[0].lower()

    if len(arg) > 1:
        return 'Error: please enter one letter only.'

    counts = Counter()
    for file in AbsPath(__file__).glob('../layouts/*.json'):
        ll = memory.parse_file(file)

        if arg not in ll.keys:
            continue

        finger = ll.keys[arg].finger

        pairs = [x for x in ll.keys if ll.keys[x].finger == finger]
        counts.update(pairs)

    counts.pop(arg)

    res = counts.most_common()
    return '\n'.join(['```'] + [f'{(x[0] + arg).upper()} {x[1]:>3}' for x in res[:15]] + ['```'])