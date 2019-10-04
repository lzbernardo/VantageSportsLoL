# Ability correction is pretty hazy in terms of what it considers errors and how it fixes them.
# The goal is to smooth things out in order to make it easier to detect meaningful changes.
# However, the current implementation is far from perfect.
# 1. If the signal is noisy, we generally change everything to the max of the signal range.
# This tends to inflate the brightness numbers and adjust things upwards too aggressively.
# 2. If the signal is wrong during a period of change (like an incorrect value during 
# an increasing trend), we don't deal with that correctly. We only detect errors when
# we expect the signal to be flat. 
class AbilityCorrect:
    def __init__(self, big_drop=20, debug=False):
        self.debug = debug
        self.global_maxes = {}

        # the algorithm looks ahead and behind the current value this many places,
        # for a total window size of (2*window_size+1) to reduce jitter
        self.window_size = 4

        # We only allow brightness drops of this magnitude or larger. Smaller drops
        # are declared as errors
        self.big_drop = big_drop

    def fix(self, all_states, index):
        did_fix = False
        for key in all_states[index]["brightness"].keys():
            if self.fixStat(all_states, index, key):
                did_fix = True
        return did_fix

    def fixStat(self, all_states, index, key):
        if index == 0:
            return False

        prev = all_states[index-1]["brightness"][key]
        cur = all_states[index]["brightness"][key]

        # Calculate global maxes for each key. Useful for the next step when finding changes
        if key not in self.global_maxes or prev > self.global_maxes[key]:
            if self.debug:
                print "Global max for {0} = {1}".format(key, prev)
            self.global_maxes[key] = prev

        needs_fix = False

        # The only brightness drops should be fairly big, representing an ability usage
        if cur < prev and cur > prev-self.big_drop:
            if self.debug:
                print "Drop isn't big enough in {0} at frame {1}: Found {2}, but saw {3} before".format(
                    key, all_states[index]["frame_num"], cur, prev)
            needs_fix = True

        # Clean up jitters. If most of the values before and after are the same, yet the current
        # value is different, correct the current value
        # This metric is somewhat controversial and could be a prime candidate for something better.
        # Having some threshold for change, or detecting a trend would probably be better
        elif cur != prev and index >= self.window_size and index < len(all_states) - self.window_size:
            window_start = index-self.window_size
            window_end = index+self.window_size

            equal_past_values = filter(lambda x: x["brightness"][key] == prev, all_states[window_start:index])
            equal_future_values = filter(lambda x: x["brightness"][key] == prev, all_states[index+1:window_end])

            if len(equal_past_values) >= self.window_size/2 and len(equal_future_values) >= self.window_size/2:
                if self.debug:
                    print "Brightness jitter in {0} at frame {1}: Found {2}, but saw {3} before and after".format(
                        key, all_states[index]["frame_num"], cur, prev)
                needs_fix = True
            
        if needs_fix:
            all_states[index]["brightness"][key] = prev
        
        return needs_fix
