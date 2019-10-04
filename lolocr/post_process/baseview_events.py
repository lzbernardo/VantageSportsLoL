#

def baseview_events_from_change_events(frame):
    baseview_events = []
    for event in frame.get('events', []):
        match_sec = frame.get('match_sec')
        if match_sec == None or match_sec < 0:
            continue
        (key, eType, pct) = event.get('key'), event.get('type'), event.get('pct')
        if key in ['q', 'w', 'e', 'r'] and eType == 'CHANGE' and pct < -30:
            baseview_events.append({
                'attacker': str(frame.get('summoner_id', -1)),
                'ability': key.upper(),
                'landed': False,
                'seconds': frame.get('match_sec')
            })
    return baseview_events


def add_baseview_events(frame_states, debug=False):
    for frame in frame_states:
        baseview_events = baseview_events_from_change_events(frame)
        if len(baseview_events) > 0:
            frame['baseview_events'] = baseview_events


def extract_baseview_events(frame_states):
    baseview_events = []
    for state in frame_states:
        baseview_events.extend(state.get('baseview_events', []))
    return baseview_events
