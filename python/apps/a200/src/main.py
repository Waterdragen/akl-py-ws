import os
import json
import copy

from . import layout
from . import analyzer

from typing import TYPE_CHECKING

current_directory = os.path.dirname(os.path.abspath(__file__))  # ./a200/src
parent_directory = os.path.dirname(current_directory)  # ./a200
os.chdir(parent_directory)

if TYPE_CHECKING:
    from apps.consumers import A200Consumer

JSON = dict[str, any]


def get_layout_percent(item: JSON, metric: str, results: JSON):
    wins = 0
    for result in results['data']:
        if item['metrics'][metric] > result['metrics'][metric]:
            wins += 1
    return wins / len(results['data'])


def sort_results(results: JSON, config: JSON):
    # calulate sort criteria
    for item in results['data']:
        for sort in config['sort']:
            percent = get_layout_percent(item, sort, results)
            if str(config['sort'][sort])[0] == '-':
                value = 1 - percent
            else:
                value = percent

            item['sort'] += value * abs(config['sort'][sort])

    # sort
    if config['sort'] == 'name':
        results['data'] = sorted(results['data'], key=lambda x: x['name'].lower(), reverse=config['sort-high'])
    else:
        results['data'] = sorted(results['data'], key=lambda x: x['sort'], reverse=config['sort-high'])


def flatten(section: JSON):
    res = {}
    for item in section:
        if type(section[item]) == bool:
            res[item] = section[item]
        else:
            res.update(flatten(section[item]))

    return res


def get_states(section: JSON):
    if type(section) == bool:
        return [section]

    states = []
    for item in section:
        states += get_states(section[item])

    return states


def find_section(section: JSON, target: str):
    if type(section) != dict:
        return []

    matches = []
    for item in section:
        if item == target:
            matches.append(section)

        matches += find_section(section[item], target)

    return matches


def set_states(section: JSON, new_state: bool):
    for item in section:
        if type(section[item]) == bool:
            section[item] = new_state
        else:
            set_states(section[item], new_state)


def parse_command(args: list[str]) -> tuple[str | None, list[str]]:
    _a200 = args[0]
    action: str | None = args[1] if len(args) > 1 else None
    action_args: list[str] = args[2:] if len(args) > 2 else []
    return action, action_args


class A200:
    def __init__(self, consumer: "A200Consumer"):
        self.consumer = consumer
        self.console_messages: list[str] = []

    def init_config(self):
        config: dict = json.load(open(os.path.join('src', 'static', 'config-init.json'), 'r'))
        layouts = layout.load_dir(config['layoutdir'])

        for keys in layouts:
            config['layouts'][keys['name'].lower()] = True

        self.consumer.get_session()
        self.consumer.update_config(config)

        return config

    def get_results(self, config: JSON):

        # get/create results cache
        cachefile: str = "cached-" + config["datafile"]
        if not (cache := self.consumer.get_cache(cachefile)):
            cache = {
                'file': config['datafile'],
                'data': {}
            }

        layouts = layout.load_dir(config['layoutdir'])
        data = json.load(open(os.path.join(config['datadir'], config['datafile'] + '.json'), 'r'))

        results = {
            'file': data['file'],
            'data': []
        }

        for keys in layouts:
            item: dict = {
                'name': keys['name'],
                'file': keys['file'],
                'sort': 0,
                'metrics': {},
            }

            # add key if it doesn't exist or contains a hash mismatch
            if (
                    not keys['name'] in cache['data'] or
                    keys['hash'] != cache['data'][keys['name']]['hash']
            ):
                cache['data'][keys['name']] = {
                    'hash': keys['hash']
                }

            # get stats
            if not config['thumb-space'] in cache['data'][item['name']]:
                cache['data'][item['name']][config['thumb-space']] = analyzer.get_results(keys, data, config)

            item['metrics'] = cache['data'][item['name']][config['thumb-space']]

            results['data'].append(item)

        sort_results(results, config)

        # update cache
        self.consumer.update_cache(cache, cachefile)

        return results

    def run_args(self, action: str | None, args: list[str]) -> dict | None:

        # open/init config
        if not (config := self.consumer.get_config()):
            config = self.init_config()

        # parse args
        if action in ['view', 'vw']:
            config['single-mode']['active'] = len(args) != 0
            config['single-mode']['layouts'] = [x.lower() for x in args]

        elif action in ['toggle', 'tg', 'tc', 'tl']:

            config['single-mode']['active'] = False

            # parse target and axis
            axis = ""
            if action in ['toggle', 'tg']:
                if len(args) == 0:
                    self.push(f"try:\n"
                              f"./a200 toggle column [column]\n"
                              f"./a200 toggle layout [layout]")
                    return None

                if args[0] in ['column', 'c']:
                    axis = 'columns'
                elif args[0] in ['layout', 'l']:
                    axis = 'layouts'
                targets = args[1:]
            else:
                if action == 'tc':
                    axis = 'columns'
                elif action == 'tl':
                    axis = 'layouts'
                targets = args[0:]

            # recursively find and set states for each target
            for target in targets:
                if axis == 'layouts':
                    target = target.lower()

                if target in ['all', 'a']:
                    set_states(config[axis], True not in get_states(config[axis]))
                else:
                    section = find_section(config[axis], target)[0]
                    if type(section[target]) == bool:
                        section[target] = not section[target]
                    else:
                        set_states(section[target], True not in get_states(section[target]))

        elif action in ['sort', 'st']:

            config['single-mode']['active'] = False

            config['sort'] = {}

            count = 0
            total_percent = 0

            for arg in args:
                # sorting direction
                if arg in ['high', 'h']:
                    config['sort-high'] = True
                elif arg in ['low', 'l']:
                    config['sort-high'] = False
                # parse metric string
                else:
                    if '%' not in arg:
                        if arg[0] == '-':
                            arg = ('-', arg[1:])
                        else:
                            arg = ('', arg)
                        count += 1
                    else:
                        arg = arg.split('%')
                        total_percent += abs(float(arg[0]))

                    config['sort'][arg[1]] = arg[0]

            # calculate percent per unassigned metric
            if count:
                percent_left = (100 - total_percent) / count
            else:
                percent_left = 0

            # allocate percents and convert to float
            for item in config['sort']:
                if config['sort'][item] in ['', '-']:
                    config['sort'][item] += str(percent_left)
                config['sort'][item] = float(config['sort'][item]) / 100

        elif action in ['filter', 'fl']:

            config['single-mode']['active'] = False

            config['filter'] = {}

            # parse metric string
            for arg in args:
                arg = arg.split('%')

                config['filter'][arg[1]] = arg[0]

            # convert to float
            for item in config['filter']:
                config['filter'][item] = float(config['filter'][item]) / 100

        elif action in ['thumb', 'tb']:

            if args[0].upper() in ['LT', 'RT', 'NONE', 'AVG']:
                config['thumb-space'] = args[0].upper()

        elif action in ['data', 'dt']:

            if os.path.isfile(os.path.join(config['datadir'], args[0] + '.json')):
                config['datafile'] = args[0]

        elif action in ['theme', 'tm']:

            if os.path.isfile(os.path.join(config['themedir'], args[0] + '.json')):
                config['theme'] = args[0]

        elif action in ['reset']:

            config = self.init_config()

        elif action in ['config', 'cs', 'cl']:

            self.push(f"Unsupported command: `{action}`")

        elif action in ['cache', 'cc']:

            self.consumer.clear_cache()

        elif action in ['help', 'hp', 'h', '?']:

            config['single-mode']['active'] = False

            args_help = json.load(open('./src/static/args-help.json', 'r'))
            self.push(args_help['desc'])
            for item in args_help['actions']:
                item_args = ' | '.join([x for x in item['args']])

                self.push(
                    '   ',
                    (item['name'] + ' | ' + item['alias']).ljust(24, ' '),
                    item_args.ljust(24, ' '),
                    item['desc'],
                )
                self.push()

            return None

        elif action is None:

            config['single-mode']['active'] = False

        return config

    def show_results(self, results: JSON, config: JSON):
        # print metadata
        self.push(results['file'].upper())

        if config['filter']:
            self.push("filter by:", end='   ')
            for filter in config['filter']:
                self.push(
                    filter,
                    "{:.2%}".format(config['filter'][filter]),
                    end='   '
                )
            self.push()

        if config['sort']:
            self.push("sort by:", end='   ')
            for sort in config['sort']:
                self.push(
                    sort,
                    "{:.0%}".format(config['sort'][sort]),
                    end='   '
                )
            self.push()

        # self.send(("sort by " + config['sort'].upper() + ":").ljust(22, ' '), end=' ')
        self.push(("thumb: " + config['thumb-space']).ljust(22, ' '), end=' ')

        # print column names
        for metric, value in flatten(config['columns']).items():
            if value:
                self.push(metric.rjust(8, ' '), end=' ')
        self.push()

        # get filters
        filters = []
        for name, val in config['filter'].items():
            filter = {
                'name': name,
                'dir': val // abs(val),
                'cutoff': abs(val)
            }

            filters.append(filter)

        # print rows
        for item in results['data']:

            if item['name'].lower() not in config['layouts']:
                config['layouts'][item['name'].lower()] = True

            # ignore hidden layouts
            if not config['layouts'][item['name'].lower()]:
                continue

            # ignore filter layouts
            for filter in filters:
                name = filter['name']
                dir = filter['dir']
                cutoff = filter['cutoff']

                if dir * (item['metrics'][name] - cutoff) < 0:
                    break
            else:
                # print layout stats
                self.push((item['name'] + '\033[38;5;250m' + ' ').ljust(36, '-') + '\033[0m', end=' ')
                for metric, value in flatten(config['columns']).items():
                    if value:
                        self.print_color(item, metric, results, config, metric not in ['roll-rt', 'oneh-rt'])
                self.push()

    def print_layout(self, results: JSON, config: JSON):
        self.push(results['file'].upper())
        self.push(("thumb: " + config['thumb-space']).ljust(22, ' '))

        for item in [item for item in results['data'] if config['layouts'][item['name'].lower()]]:

            self.push()

            # header
            self.push(item['name'])
            self.push(layout.pretty_print(item['file'], config))
            self.push()

            self.push('Trigrams')
            self.push('========')

            # alternation
            self.push('Alternates -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'alternate', results, config)
            self.push()

            # rolls
            self.push('Rolls -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'roll', results, config)
            self.push('In:', end=' ')
            self.print_color(item, 'roll-in', results, config),
            self.push('Out:', end=' ')
            self.print_color(item, 'roll-out', results, config)
            self.push('Ratio:', end=' ')
            self.print_color(item, 'roll-rt', results, config, False)
            self.push()

            # onehands
            self.push('Onehands -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'onehand', results, config)
            self.push('In:', end=' ')
            self.print_color(item, 'oneh-in', results, config),
            self.push('Out:', end=' ')
            self.print_color(item, 'oneh-out', results, config)
            self.push('Ratio:', end=' ')
            self.print_color(item, 'oneh-rt', results, config, False)
            self.push()

            # redirects
            self.push('Redirects -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'redirect', results, config)
            self.push()

            # unknown
            if item['metrics']['unknown'] > 0:
                self.push('Unknown -'.rjust(12, ' '), end=' ')
                self.push('Total:', end=' ')
                self.print_color(item, 'unknown', results, config)
                self.push()

            self.push()

            # sfb/dsfb/sfT/sfR
            self.push('Same Finger')
            self.push('===========')

            self.push('SFB -'.rjust(12, ' '), end='')
            self.print_color(item, 'sfb', results, config)
            self.push('DSFB -'.rjust(12, ' '), end='')
            self.print_color(item, 'dsfb', results, config)
            self.push()

            self.push('SFT -'.rjust(12, ' '), end='')
            self.print_color(item, 'sfT', results, config)
            self.push('SFR -'.rjust(12, ' '), end='')
            self.print_color(item, 'sfR', results, config)
            self.push()

            self.push()

            # finger use
            self.push("Finger Use")
            self.push("==========")

            self.push('Left -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'LTotal', results, config)
            for finger in ['LP', 'LR', 'LM', 'LI']:
                self.push(finger + ':', end=' ')
                self.print_color(item, finger, results, config)
            self.push()

            self.push('Right -'.rjust(12, ' '), end=' ')
            self.push('Total:', end=' ')
            self.print_color(item, 'RTotal', results, config)
            for finger in ['RP', 'RR', 'RM', 'RI']:
                self.push(finger + ':', end=' ')
                self.print_color(item, finger, results, config)
            self.push()

            if config['thumb-space'] != 'NONE':
                self.push('Thumb -'.rjust(12, ' '), end=' ')
                self.push('Total:', end=' ')
                self.print_color(item, 'TB', results, config)
                self.push()

            self.push()

            # row use
            self.push("Row Use")
            self.push("=======")

            self.push('Top -'.rjust(12, ' '), end=' ')
            self.print_color(item, 'top', results, config)
            self.push('Home -'.rjust(12, ' '), end=' ')
            self.print_color(item, 'home', results, config)
            self.push('Bottom -'.rjust(12, ' '), end=' ')
            self.print_color(item, 'bottom', results, config)
            self.push()

    def print_color(self, item: JSON, metric: str, data: JSON, config: JSON, is_percent: bool = True):
        # get percentage of layouts worse
        percent = get_layout_percent(item, metric, data)

        # get string

        if is_percent:
            string = "{:.2%}".format(item['metrics'][metric]).rjust(6, ' ') + '  '
        else:
            string = "{0:.2f}".format(item['metrics'][metric]).rjust(6, ' ') + ' '

        # color printing based on percentage
        colors = json.load(open(os.path.join(config['themedir'], config['theme'] + '.json'), 'r'))['colors']
        if percent > .9:
            self.push('\033[38;5;' + colors['highest'] + 'm' + string + '\033[0m', end=' ')
        elif percent > .7:
            self.push('\033[38;5;' + colors['high'] + 'm' + string + '\033[0m', end=' ')

        elif percent < .1:
            self.push('\033[38;5;' + colors['lowest'] + 'm' + string + '\033[0m', end=' ')
        elif percent < .3:
            self.push('\033[38;5;' + colors['low'] + 'm' + string + '\033[0m', end=' ')
        else:
            self.push('\033[38;5;' + colors['base'] + 'm' + string + '\033[0m', end=' ')

    def main(self, args: list[str]) -> str:
        action, action_args = parse_command(args)
        config = self.run_args(action, action_args)
        if config is None:
            return self.gather_console_log()

        if config['single-mode']['active']:
            layout_config = copy.deepcopy(config)
            layout_config['layouts'] = {item: False for item in layout_config['layouts']}

            for layout_name in layout_config['single-mode']['layouts']:
                layout_config['layouts'][layout_name] = True

            results = self.get_results(layout_config)
            self.print_layout(results, layout_config)
        else:
            results = self.get_results(config)
            self.show_results(results, config)

        # Update config
        self.consumer.update_config(config)

        return self.gather_console_log()

    # Push the string to the console messages
    def push(self, *messages: str, end="\n"):
        text_data = "".join(messages) + end
        self.console_messages.append(text_data)

    def gather_console_log(self) -> str:
        return "".join(self.console_messages)
