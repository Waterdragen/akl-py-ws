import os
import json
from jellyfish import damerau_levenshtein_distance as lev

from ..core.keyboard import Layout, Position
from ..core.resource import AbsPath


def get(name: str) -> Layout | None:
    file = f'../layouts/{name}.json'

    if not os.path.exists(file):
        return None

    return parse_file(file)


def find(name: str) -> Layout:
    file = f'../layouts/{name}.json'

    if not os.path.exists(file):
        names = [AbsPath.get_basename(x) for x in AbsPath(__file__).glob('../layouts/*.json')]
        names = sorted(names, key=lambda x: len(x))

        closest = min(names, key=lambda x: lev((''.join(y for y in x.lower() if y in name)), name))

        file = f'../layouts/{closest}.json'

    return parse_file(file)

def parse_file(file: str) -> Layout:
    with AbsPath(__file__).open(file, 'r') as f:
        data = json.load(f)

        keys = {
            k: Position(
                row=v["row"],
                col=v["col"],
                finger=v["finger"]
            ) for k, v in data["keys"].items()
        }

        ll = Layout(
            name=data["name"],
            user=data["user"],
            board=data["board"],
            keys=keys,
        )

    return ll
