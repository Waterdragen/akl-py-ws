import random

from apps.cmini.core.resource import CminiData


def exec(user: CminiData):
    return random.choice([
        'Yes', 'Count on it',
        'No doubt',
        'Absolutely', 'Very likely',
        'Maybe', 'Perhaps',
        'No', 'No chance', 'Unlikely',
        'Doubtful', 'Probably not'
    ])
