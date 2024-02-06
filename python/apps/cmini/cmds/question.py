import json
import requests

from apps.cmini.core.resource import CminiData

RESTRICTED = False

def exec(user: CminiData):
    file = 'https://story-shack-cdn-v2.glitch.me/generators/random-question-generator'

    req = requests.get(file)
    res = json.loads(req.text)

    return res['data']['name']
