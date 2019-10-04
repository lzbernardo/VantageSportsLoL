import json, sys

# get_first_skill_level_ups reads through the timeline events in the match
# details and finds the first occurrence of each skill-slot's level up, skipping
# those that occur within the first 60 seconds (since those may be for reasons
# that OCR can't detect) and returns a map from skill -> match seconds that the
# event occurred.
def get_first_skill_level_ups(match_details, participant_id, min_seconds=60):
    level_secs = {}

    slot_names = ['', 'q', 'w', 'e', 'r']
    for frame in match_details.get('timeline', {}).get('frames', []):
        for event in frame.get('events', []):
            if event.get('participantId') != participant_id:
                continue
            if event.get('eventType') != 'SKILL_LEVEL_UP':
                continue
            seconds = event.get('timestamp') / 1000
            if seconds < min_seconds:
                continue
            slot_name = slot_names[event.get('skillSlot', 0)]
            if not level_secs.get(slot_name):
                level_secs[slot_name] = seconds

    return level_secs

# get_first_ability_video_seconds iterates through the ocr frame states and 
# finds the first occurrence of each skill's event indicating its acquisition
# and records the video time of that event. The map of skill -> video seconds
# is returned.
def get_first_ability_video_seconds(frames, participant_id, min_seconds=60):
    acq_secs = {}

    for f in frames:
        if f.get('pov_champ') != participant_id:
            continue
        if f.get('video_sec') < min_seconds:
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
# match time.
def get_video_second_offset(match_details, ocr_frames, participant_id):
    ocr_secs = get_first_ability_video_seconds(ocr_frames, participant)
    match_secs = get_first_skill_level_ups(match_details, participant)

    diffs = []
    for key, match_secs in api_secs.items():
        video_secs = acq_secs.get(key)
        if not video_secs:
            continue
        diffs.append(match_secs - video_secs)
    return sum(diffs) / len(diffs)


# load is a debug helper function for loading a json file
def load(path):
    f = open(path)
    j = json.loads(f.read())
    f.close()
    return j

if __name__ == '__main__':

    frames = load(sys.argv[1])
    match_details = load(sys.argv[2])
    participant = int(sys.argv[3])

    print get_video_second_offset(frames, match_details, participant)
