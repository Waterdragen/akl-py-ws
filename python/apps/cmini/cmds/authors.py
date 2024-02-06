import json

from apps.cmini.core.resource import CminiData, AbsPath


def exec(user: CminiData):

    with AbsPath(__file__).open('../authors.json', 'r') as f:
        authors = json.load(f)

    lines = ['Layout Creators:']
    lines.append('```')
    lines += list(sorted(authors.keys(), key=lambda x: x.lower()))
    lines.append('```')

    return '\n'.join(lines)