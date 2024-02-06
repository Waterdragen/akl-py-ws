import json
from jellyfish import damerau_levenshtein_distance as lev

from ..core.resource import AbsPath


def get_id(name: str) -> int:
    with AbsPath(__file__).open('../authors.json', 'r') as f:
        authors = json.load(f)

    if name in authors:
        return int(authors[name])
    else:
        names = sorted(authors.keys(), key=lambda x: len(x))
        closest = min(names, key=lambda x: lev((''.join(y for y in x.lower() if y in name)), name))

        return int(authors[closest])


def get_name(id: int) -> str:
    if id == 0:
        return "Guest"

    with AbsPath(__file__).open('../authors.json', 'r') as f:
        authors = json.load(f)

    names = [k for k, v in authors.items() if int(v) == id]

    if names:
        return names[0]
    else:
        return 'unknown'