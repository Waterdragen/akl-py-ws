import os

from ..core.resource import CminiData, AbsPath
from ..util import parser

def exec(user: CminiData):
    name = parser.get_arg(user).lower()

    corpora = [os.path.basename(x) for x in AbsPath(__file__).glob('../corpora/*')]
    print("Corpora: " + str(corpora))

    if not name:
        return '\n'.join(['```', 'List of Corpora:'] + [f'- {x}' for x in list(sorted(corpora))] + ['```'])

    if name not in corpora:
        return f'The corpus `{name}` doesn\'t exist.'

    user.set_corpus(name)

    return f'Your corpus preference has been changed to `{name}`.'

def use():
    return 'corpus [corpus_name]'

def desc():
    return 'set your preferred corpus'
