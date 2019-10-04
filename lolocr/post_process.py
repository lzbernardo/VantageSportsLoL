# this script is the "launcher" for various post-processing wrappers.

import argparse, json, os, sys

from util import read_json, write_json
from post_process.add_match_details import add_match_details
from post_process.baseview_events import add_baseview_events
from post_process.correct_states import correct_states
from post_process.state_change import find_global_maxes, add_change_events

def main():
    parser = argparse.ArgumentParser(description='LolOCR post-processing wrapper.')
    parser.add_argument('-i', '--input', help='input (json) path', required=True)
    parser.add_argument('-o', '--output', help='output (json) path', required=True)
    parser.add_argument('-s', '--steps', nargs='+', help='post_process steps to run', required=True)
    parser.add_argument('-m', '--match_details', help='path to match details json file')
    parser.add_argument('-p', '--participant_id', help='the participant id', default=0, type=int)
    parser.add_argument('-u', '--summoner_id', help='the summoner id', default=0, type=int)
    parser.add_argument('-v', '--verbose', help='verbose debugging info', dest='verbose', action='store_true')
    parser.add_argument('--pretty', help='if true, indents the output', dest='pretty', action='store_true')
    args = parser.parse_args()

    obj = read_json(args.input)
    
    if type(obj) == list:
        obj = {
            'steps': [],  # the process steps this game has had run
            'states': obj
        }

    for step in args.steps:
        if step == 'correct':
            do_correct(obj, debug=args.verbose)

        elif step == 'maxes':
            do_maxes(obj, debug=args.verbose)

        elif step == 'changes':
            do_changes(obj, debug=args.verbose)

        elif step == 'align':
            do_match_align(obj, args, debug=args.verbose)

        elif step == 'baseview':
            do_baseview(obj, args, debug=args.verbose)

        else:
            print 'unknown step:', step
            sys.exit(1)

    write_json(args.output, obj, args.pretty)


def do_correct(obj, debug=False):
    if debug:
        print 'correcting states...'

    obj['steps'].append('correct')
    obj['corrections'] = correct_states(obj['states'], debug=debug)

    if debug:
        print 'fixed %d health_mana errors' % len(obj['corrections']['health_mana'])


def do_maxes(obj, debug=False):
    if 'correct' not in obj['steps']:
        do_correct(obj, debug=debug)
    if debug:
        print 'computing maxes...'
    
    obj['steps'].append('maxes')
    obj['brightness_maxes'] = find_global_maxes(obj['states'])
    return obj['brightness_maxes']


def do_change_events(obj, debug=False):
    maxes = obj.get('brightness_maxes')
    if maxes == None:
        maxes = do_maxes(obj, debug=debug)

    obj['steps'].append('changes')
    add_change_events(obj['states'], max_vals=maxes)


def do_match_align(obj, args, debug=False):
    if 'changes' not in obj['steps']:
        do_change_events(obj, debug=debug)
    if debug:
        print 'adding match times...'

    if args.summoner_id < 1:
        print 'summoner_id must be greater than 0'
        sys.exit(1)
    if args.participant_id < 1 or args.participant_id > 10:
        print 'participant_id must be >= 1 and <= 10'
        sys.exit(1)
    match_details = read_json(args.match_details)

    obj['steps'].append('align')
    add_match_details(match_details, obj['states'], args.participant_id, args.summoner_id)


def do_baseview(obj, args, debug=False):
    if 'align' not in obj.get('steps'):
        do_match_align(obj, args, debug=debug)
    if debug:
        print 'computing baseview events...'

    obj['steps'].append('baseview')
    add_baseview_events(obj['states'], debug=debug)


if __name__ == '__main__':
    main()
