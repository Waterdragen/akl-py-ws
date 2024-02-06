import random

from apps.cmini.core.resource import CminiData

RESTRICTED = False

EMOJIS = [

]

def exec(user: CminiData):
    dofs = [emoji for emoji in EMOJIS
            if emoji.available
            and 'dof' in emoji.name.lower()]

    if not dofs:
        return "No dofs in here :("

    dof = random.choice(dofs)

    prefix = 'a' if dof.animated else ''

    return f'<{prefix}:{dof.name}:{dof.id}>'
