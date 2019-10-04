# class Window(object):

#     def __init__(self, frames=[], min_length=5, min_secs=0.5):
#         self._frames = frames
#         self._min_length = min_length
#         self._min_secs = min_secs
#         self._poly = None

#     # def complete(self, min_length=2, min_secs=1):
#     #     if len(self._frames) < min_length:
#     #         return False
#     #     if self.duration() < min_secs:
#     #         return False
#     #     return True

#     def duration(self):
#         end_sec = self._frames[-1]['video_sec']
#         start_sec = self._frames[0]['video_sec']
#         return end_sec - start_sec

#     def is_stable(self):
#         if len(self._frames) < self._min_length:
#             return False
#         if self.duration() < self._min_secs:
#             return False
#         return True

#     def add(self, frame):
#         self._frames.append(frame)
#         if self._poly != None:
#             #value_diff = self._frames[-1]['val'] - self._frames[0]['val']
#             #secs_diff = self._frames[-1]['video_sec'] - self._frames[0]['video_sec']
#             self._poly, self._predictor = fit(self._frames)
#             #print 'new rate', 1.0 * value_diff / secs_diff, value_diff, secs_diff
#             #self._poly[0] = 1.0 * value_diff / secs_diff

#     def predicts(self, frame, tolerance=0.05):
#         if self._poly == None:
#             self._poly, self._predictor = fit(self._frames)
#         predicted = self._predictor(frame['video_sec'])
#         actual = frame['val']
#         diff_pct = 1.0 * abs(predicted - actual) / actual
#         if diff_pct <= tolerance:
#             return True
#         print 'predicted', predicted, 'actual:', actual, '%', diff_pct
#         #print predicted, actual, diff_pct, tolerance
#         return diff_pct <= tolerance


    # def value_diff(self):
    #     end_val = self._frames[-1]['brightness'][self._key]
    #     start_val = self._frames[0]['brightness'][self._key]
    #     return end_val - start_val

    # def rate(self):
    #     if len(self._frames) < 2:
    #         return None
    #     secs = [ f['video_sec'] for f in self._frames ]
    #     vals = [ f['brightness'][key] for f in self._frames ]
    #     #A = np.vstack([secs, np.ones(len(secs))]).T
    #     #slope, coeff = np.linalg.lstsq(A, vals)
    #     slope, fit = np.polyfit(secs, vals, 1)

    #     return 1.0 * self.value_diff() / self.duration()

    # def fits(self, frame):
    #     candidate = self._frames + [frame]
    #     new_slope, new_fit = fit(candidate)
    #     #print new_slope, new_fit

# def fit(frames):
#     secs = [ f['video_sec'] for f in frames ]
#     vals = [ f['val'] for f in frames ]
#     z = np.polyfit(secs, vals, 1)
#     return z, np.poly1d(z)


# # To determine rate changes, we look at a sliding window of values to determine
# # a rate of change. When that rate changes, we record it.
# def rate_changes(frames, min_window_length, min_window_seconds):
#     rates = {}
#     window = None

#     for f in frames:
#         val_frame = {
#             'video_sec': f['video_sec'],
#             'frame_num': f['frame_num'],
#             'val': f['brightness']['q'],
#         }
#         if not window:
#             window = Window()
#         if not window.is_stable():
#             window.add(val_frame)
#             continue
#         elif window.predicts(val_frame):
#             window.add(val_frame)
#             continue

#         print "change at", val_frame['video_sec'], 'after', len(window._frames), 'frames'
#         print " "
#         window = Window([val_frame])


#         # last.append(val_frame)
#         # # if the window rejects the frame, then something significant happened.
#         # while len(last) > window_length:
#         #     last = last[1:]
#         # while last[-1]['video_sec'] - last[0]['video_sec'] > window_seconds:
#         #     last = last[1:]

#         # slope, rs = fit(last):


#         # for key, val in f['brightness'].items():
#         #     val_frame = {
#         #         'video_sec': f['video_sec'],
#         #         'frame_num': f['frame_num'],
#         #         'val': val,
#         #     }
#         #     window = windows.get(key)
#         #     if not window:
#         #         window = Window()

#         #     if not window.fits(val_frame):
#         #         pass
#         #         # create a new window here, and emit the old window.
#         #         window.add(val_frame)

#         #     window = windows.get(key, [])

#         #     if not is_window_complete(window, min_window_length, min_window_seconds):

#         #         #add existing frame if compatible.
#         #         pass
#         #     else:
#         #         rate = get_window_rate(window)
#         #         # if rate predicts val, add frame to window
#         #         #    add it
#         #         else:
#         #             start a new window

#         # if add_to_window(event, window, rate):
#         #     pass
#         # if can_add(window, event):
#         #     window = 
#         # window, rate, change = add_to_window(e, key, val, windows, rates)
#         # if change != None:
#         #     changes.append(change)

#         # if last != None:
#         #     #sec_diff = e['video_sec'] - last['video_sec']
#         #     for key, val in e['brightness'].items():
#         #         diff = val - last['brightness'][key]
#         #         rate = diff / sec_diff
#         #         e['brightness'][key + '_rate'] = rate
#         #         if rate < -30:
#         #             print key, "decrease", e['video_clock']
#         #             uses[key] = uses.get(key, 0) + 1
#         #     if sec_diff > 3:
#         #         print "life gap at", e['video_clock']
#         # last = e

# def is_window_complete(window, min_length=1, min_sec=1):
#     if len(window) < min_length:
#         return True
#     duration = window[-1]['video_sec'] - window[0]['video_sec']
#     if duration < min_sec:
#         return True

# def add_to_window(event, key, val, windows, rates):
#     window, rates = windows.get(key, []), rates.get(key)
#     if len(window) == 0:
#         windows[key] = [event]
#         rates[key] = None
#         return None

#     if rate != None:
#         if can_predict(val, rate, window):
#             window[key] = window + [event]
#             return None
#         else:
#             print key, val, "CHANGE"
#             return "CHANGE"

# def histogram(events, key):
#     hist = {}
#     for e in events:
#         val = e['brightness'][key]
#         hist_key = '%d' % val
#         hist[hist_key] = hist.get(hist_key, 0) + 1
#     return hist
