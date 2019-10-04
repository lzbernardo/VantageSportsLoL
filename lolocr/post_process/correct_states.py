import argparse
import json

from health_mana_correct import HealthManaCorrect
from brightness_correct import AbilityCorrect
from location_correct import LocationCorrect

def correct_states(states, debug=False):
    healthManaCorrect = HealthManaCorrect(debug=debug)
    abilityCorrect = AbilityCorrect(debug=debug)
    locationCorrect = LocationCorrect(debug=debug)
    corrections = {
        "health_mana": {
            "count":0,
            "frames":[],
        },
        "ability": {
            "count":0,
            "frames":[],
        },
        "location": {
            "count":0,
            "frames":[],
        },
    }

    for i in range(0, len(states)):
        if healthManaCorrect.fix(states, i):
            corrections["health_mana"]["count"] += 1
            corrections["health_mana"]["frames"].append(states[i]["frame_num"])
        if abilityCorrect.fix(states, i):
            corrections["ability"]["count"] += 1
            corrections["ability"]["frames"].append(states[i]["frame_num"])
        if locationCorrect.fix(states, i):
            corrections["location"]["count"] += 1
            corrections["location"]["frames"].append(states[i]["frame_num"])

    # Add the global brightness stat into the every state
    for i in range(0, len(states)):
        states[i]["brightness_max"] = abilityCorrect.global_maxes

    return corrections

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Correct errors in ocr states')
    parser.add_argument('-i','--input', help='input json', required=True)
    parser.add_argument('-o','--output', help='output json')
    parser.add_argument('--pretty', help='if true, indents the output', dest='pretty', action='store_true')
    parser.add_argument('-v','--verbose', action='store_true', help='verbose debugging info')
    args = parser.parse_args()

    data = None
    with open(args.input) as data_file:
        data = json.load(data_file)

        corrections = correct_states(data, debug=args.verbose)
        print json.dumps({"corrections":corrections}, indent=4)

    if args.output:
        with open(args.output, "w") as out_file:
            if args.pretty:
                out_file.write(json.dumps(data, indent=4))
            else:
                out_file.write(json.dumps(data))

