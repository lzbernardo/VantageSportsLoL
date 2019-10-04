import json, sys


# get_first_skill_level_ups reads through the timeline events in the match
# details and finds the first occurrence of each skill-slot's level up and
# returns a map from skill -> match seconds that the event occurred.
def get_first_skill_level_ups(match_details, participant_id):
    level_secs = {}

    slot_names = ['', 'q', 'w', 'e', 'r']
    for frame in match_details.get('timeline', {}).get('frames', []):
        for event in frame.get('events', []):
            if event.get('participantId') != participant_id:
                continue
            if event.get('eventType') != 'SKILL_LEVEL_UP':
                continue
            slot_name = slot_names[event.get('skillSlot', 0)]
            if not level_secs.get(slot_name):
                level_secs[slot_name] = event['timestamp'] / 1000

    return level_secs


# get_first_ability_video_seconds iterates through the ocr frame states and 
# finds the first occurrence of each skill's event indicating its acquisition
# and records the video time of that event. The map of skill -> video seconds
# is returned.
def get_first_ability_video_seconds(frames, participant_id):
    acq_secs = {}

    for f in frames:
        if f.get('pov_champ') != participant_id:
            continue
        for e in f.get('events', []):
            key = e.get('key')
            if not key in ['q', 'w', 'e', 'r']:
                continue
            if acq_secs.get(key):
                continue
            if e.get('pct', 0) > 40:
                acq_secs[key] = f['video_sec']

    return acq_secs


# get_video_second_offset takes a map of skill acquisitions from the match
# details (containing match seconds) and ocr frame states (contining video 
# seconds) and computed the average offset between events for the same skill,
# which represents the amount of clock difference between the video and the
# match time. An offset of 5.2 means that the match time is 5.2 seconds ahead
# of the video time.
def get_video_second_offset(match_details, ocr_frames, participant_id):
    ocr_secs = get_first_ability_video_seconds(ocr_frames, participant_id)
    match_secs = get_first_skill_level_ups(match_details, participant_id)

    diffs = []
    for key, match_secs in match_secs.items():
        video_secs = ocr_secs.get(key)
        if not video_secs:
            continue
        diffs.append(match_secs - video_secs)
    return sum(diffs) / len(diffs)

# add_match_times computes the seconds offset between the match time and the
# video time, and uses that to add a "match_sec" field to each frame containing
# a positive match time in seconds.
def add_match_times(match_details, ocr_frames, participant_id):
    offset = get_video_second_offset(match_details, ocr_frames, participant_id)

    for f in ocr_frames:
        match_sec = f['video_sec'] + offset
        if match_sec >= 0:
            f['match_sec'] = round(match_sec, 1)
    return ocr_frames


# add_summoner_id finds the summoner id that belongs to the provided participant
# id. note that the summoner_id is only in the player object for ranked matches,
# and not for custom or normal matches.
def get_summoner_id(match_details, participant_id):
    summoner_id = -1
    for pi in match_details.get('participantIdentities', []):
        if pi.get('participantId', -1) == participant_id:
            return pi.get('player', {}).get('summonerId', 0)


def add_summoner_id(ocr_frames, summoner_id):
    for frame in ocr_frames:
        if summoner_id > 0:
            frame['summoner_id'] = summoner_id


def add_match_details(match_details, ocr_frames, participant_id, summoner_id):
    if summoner_id < 1:
        summoner_id = get_summoner_id(match_details, participant_id)
    add_summoner_id(ocr_frames, summoner_id)
    add_match_times(match_details, ocr_frames, participant_id)
