# this script extracts elements from frame states into a keyed object. this is
# the format expected by the baseview ingester.

import argparse, json, os, sys

from util import read_json, write_json

def main():
    parser = argparse.ArgumentParser(description='LolOCR post-processing wrapper.')
    parser.add_argument('-i', '--input', help='input (json) path', required=True)
    parser.add_argument('-o', '--output', help='output (json) path', required=True)
    parser.add_argument('-e', '--extractions', nargs='+', help='items to extract', required=True)
    parser.add_argument('-v', '--verbose', help='verbose debugging info', dest='verbose', action='store_true')
    parser.add_argument('--pretty', help='if true, indents the output', dest='pretty', action='store_true')
    args = parser.parse_args()

    input_obj = read_json(args.input)
    states = input_obj.get('states', [])

    output_obj = {}

    for extract in args.extractions:
        if extract == 'location':
            fn = lambda s: s.get('location', {})
            output_obj['locations'] = extract_from(states, fn)

        if extract == 'changes':
            fn = lambda s: s.get('events', [])
            output_obj['changes'] = extract_from(states, fn)

        if extract == 'baseview':
            fn = lambda s: s.get('baseview_events', [])
            output_obj['events'] = extract_from(states, fn)

    write_json(args.output, output_obj, args.pretty)


def extract_from(states, state_fn):
    items = []
    for state in states:
        items.extend(state_fn(state))
    return items


if __name__ == '__main__':
    main()
