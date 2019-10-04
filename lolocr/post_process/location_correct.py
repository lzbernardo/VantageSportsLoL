class LocationCorrect:
    def __init__(self, debug=False):
        self.debug = debug

    def fix(self, all_states, index):
        if index == 0:
            return False

        prev_x = all_states[index-1]["location"]["x"]
        prev_y = all_states[index-1]["location"]["y"]
        cur_x = all_states[index]["location"]["x"]
        cur_y = all_states[index]["location"]["y"]
        needs_fix = False

        # Replace any errors (-1 values) with the previous value
        if (cur_x == -1 or cur_y == -1) and (prev_x != -1 or prev_y != -1):
            if self.debug:
                print "Unknown location in {0},{1} at frame {2}. Replacing with previous value {3},{4}".format(
                    cur_x, cur_y, all_states[index]["frame_num"], prev_x, prev_y)
            needs_fix = True

        if needs_fix:
            all_states[index]["location"]["x"] = prev_x
            all_states[index]["location"]["y"] = prev_y
        
        return needs_fix
