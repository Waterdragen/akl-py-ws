import os
from importlib import import_module
from more_itertools import divide

from ..util import parser
from textwrap import wrap
from ..core.resource import CminiData, AbsPath
from ..util.consts import CMDS_PACKAGE_NAME, PACKAGE_NAME

WINDOW_SIZE = 36  # excluding indent
INDENT = "    "

def wrap_desc(desc: str) -> str:
    return "\n".join(wrap(desc, WINDOW_SIZE, initial_indent=INDENT, subsequent_indent=INDENT))


def exec(user: CminiData):
    command = parser.get_arg(user)
    commands = sorted(AbsPath.get_basename(x) for x in AbsPath(__file__).glob('../cmds/*.py'))

    print(__package__)

    if command:
        if command not in commands:
            return f"Unknown command `{command}`"

        mod = import_module(f'.cmds.{command}', package=PACKAGE_NAME)

        if hasattr(mod, 'use'):
            use = mod.use()
        else:
            use = f"{command} [args]"

        if hasattr(mod, 'desc'):
            desc = mod.desc()
        else:
            desc = "..."

        return (
            f'Help page for `{command}`:'
            f'```\n'
            f'{use}\n'
            f'{desc}\n'
            f'```'
        )

    else:
        cmds = []
        for cmd in commands:
            mod = import_module(f'.cmds.{cmd}', package=PACKAGE_NAME)

            if not all(hasattr(mod, x) for x in ['exec', 'desc', 'use']):
                continue

            cmds.append(cmd)

        lines = ['Usage: `!cmini (command) [args]`']
        lines.append('```')

        cols = divide(2, cmds)

        for row in zip(*cols):
            lines.append("".join(x.ljust(16) for x in row))

        lines.append('```')
        return '\n'.join(lines)
