import random

from apps.cmini.core.resource import CminiData

RESTRICTED = False

def exec(user: CminiData):
    return random.choices([
        '<a:wooperno:1085582045121613874>',
        '<a:wooper:1081033714043207771>',
        '<:woopwoop:1168675851592798288>'
    ], weights=[49, 49, 1])[0]
