import random

from apps.cmini.core.resource import CminiData

RESTRICTED = False

def exec(user: CminiData):
    res = random.choices(
        population=['Heads', 'Tails', 'Mail'],
        weights=[.49, .49, .02],
        k=1
    )[0]

    return f'You got `{res}`!'
