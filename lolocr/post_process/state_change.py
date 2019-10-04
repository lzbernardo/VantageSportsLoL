# state_change.py is a post_processing library capable of emitting "change 
# events" for the raw ocr data.

import os, sys

import numpy as np
        
# new_event returns a map describing the type of value change that occurred:
# CHANGE: the value changed significantly. the percentage of change (relative
#         to the previous value will be included in the 'pct' field if supplied)
# MAX:    the value reached it's maximum.
def new_event(event_type, key, pct=None):
    event = { 
        'type': event_type,
        'key': key,
    }
    if pct != None:
        event['pct'] = pct
    return event


# State keeps track of the last <window_size> values for a given signal. It
# smooths signal noise by calculating its own 'value' as the median of its last
# X readings, and can emit "change events" in the following cases:
#
# 1) the value increases significantly relative to the last value
# 2) the value decreases significantly relative to the last value
# 3) the value re-arrives at a 'maximum' value
class State(object):

    def __init__(self, key, maxVal, window_size, is_percent=False):
        self._window = []
        self._key = key
        self._last = None
        self._max = maxVal
        self._was_maxed = False
        self._is_percent = is_percent
        self._window_size = window_size

    # add adds a new value to the State's sliding window, automatically evicting
    # older entries if the window size exceeds the confiured maximum.
    def add(self, val):
        self._window.append(val)
        if len(self._window) > self._window_size:
            self._window = self._window[-1 * self._window_size:]

    # val returns the median of the samples in the state's window.
    def val(self):
        return np.median(self._window)

    # get_change_event returns a change_event if the value represented by the
    # window results in a significant event (change/max) for this state.
    def get_change_event(self, change_threshold=20):
        cur = self.val()
        last = self._last
        self._last = cur

        events = []

        # If the current value differs significantly from the last value, 
        # emit a change event.
        if last != None:
            # TODO(Cameron): we ideally want to compute this percentage relative
            # to the maximum, rather than the last. But that would require a
            # first pass before this.
            if self._is_percent:
                diff_pct = cur - last
            else:
                diff_pct = 100.0 * (cur - last) / self._max
            if abs(diff_pct) > change_threshold:
                events.append(new_event('CHANGE', self._key, diff_pct))

        # Getting really close to our maximum is considered "max'd"

        if not self._was_maxed and cur >= self._max:
            events.append(new_event('MAX', self._key))

        self._was_maxed = cur >= self._max
        return events


def evaluate_state(frame, state, val):
    state.add(val)
    return state.get_change_event(20)


def add_frame_events(frame, events):
    if len(events) == 0:
        return
    if not frame.get('events'):
        frame['events'] = []
    frame['events'].extend(events)

# add_change_events returns the change events found in the ocr frame states,
# supplied, giving each state the specified window_size.
def add_change_events(frames, max_vals={}, window_size=3):
    state_by_key = {}

    for f in frames:

        for key, val in f['brightness'].items():
            state_key = 'brightness_' + key
            state = state_by_key.get(state_key)
            if not state:
                state = State(key, max_vals.get(state_key, 100), window_size=window_size)
                state_by_key[state_key] = state
            events = evaluate_state(f, state, val)
            add_frame_events(f, events)

        for bar_name, bar in f.get('bars', []).items():
            state = state_by_key.get('bar_' + bar_name)
            if not state:
                state = State(bar_name, max_vals.get(state_key, 100), window_size=window_size, is_percent=True)
                state_by_key['bar_' + bar_name] = state
            events = evaluate_state(f, state, bar.get('percent', 0))
            add_frame_events(f, events)


def find_global_maxes(frames):
    histograms = {}

    for f in frames:
        for key, val in f['brightness'].items():
            h = histograms.get('brightness_' + key, {})
            num = h.get(val, 0)
            num += 1
            h[val] = num
            histograms['brightness_' + key] = h

        for name, bar in f.get('bars', []).items():
            h = histograms.get('bar_' + name, {})
            num = h.get(val, 0)
            num += 1
            h[val] = num
            histograms['bar_' + name] = h

    max_vals = {}
    for key, histogram in histograms.items():
        max_vals[key] = find_max(histogram)
    return max_vals


def find_max(histogram, tolerance=0.97):
    # this line looks confusing (values == keys), but because we're dealing with
    # a histogram, the keys are the unique values stored for the piece of state
    # we're tracking.
    # TODO(Cameron): we should get fancier here and use the counts in the
    # histogram (rather than a dumb tolerance) to determine the best max value.
    # It should be the 'smallest of the very common big values'.
    return int(max(histogram.keys()) * tolerance)


# Mostly for debugging, remove before submission?
if __name__ == '__main__':
    import json
    f = open(sys.argv[1])
    frames = json.loads(f.read())[:5000]
    f.close()

    max_vals = find_global_maxes(frames)
    add_change_events(frames, max_vals=max_vals)

    print json.dumps(frames, indent=2)
