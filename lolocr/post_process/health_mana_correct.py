# This attempts to correct errors in health and mana numbers by looking for obvious errors
# in the values
class HealthManaCorrect:
    def __init__(self, debug=False):
        # If the "current" value compared to the "max" * "percent" value is different
        # by more than this percent, then we declare the "current" value to be an error
        self.percentageThreshold = 0.05

        self.maxChangeThreshold = 2000

        # For detecting errors in the "max" values, this is the number of states to look
        # ahead in order to determine whether this value is correct.
        # If it's too small then we won't detect long strings of errors.
        # If it's too large then we'll have false positives and declare legitimate max
        # value fluctuations as errors (like buying and selling items)
        self.maxValueLookahead = 50
        self.debug = debug

    # Input is the full list of states, and the index to correct.
    # Each object in the list is a dictionary like:
    # {
    #    "bars": {
    #        "health": {
    #            "current": 542,
    #            "max": 542,
    #            "percent": 100
    #        },
    #        "mana": {
    #            "current": 377,
    #            "max": 377,
    #            "percent": 100
    #        }
    #    },
    #    ...
    #
    # There is other stuff in this object, but these are the only ones we care about
    #
    # This will modify the list in place, so no need to return anything
    def fix(self, allStates, index):
        didFix = False
        if self.fixMax(allStates, index, "health"):
            didFix = True
        if self.fixMax(allStates, index, "mana"):
            didFix = True
        if self.fixCur(allStates, index, "health"):
            didFix = True
        if self.fixCur(allStates, index, "mana"):
            didFix = True
        return didFix

    # We can be a bit more strict with max health and mana values because we know
    # that they don't change very often, and they tend to increase over time
    def fixMax(self, allStates, index, key):
        needsFix = False
        cur = allStates[index]["bars"][key]
        curMax = cur["max"]
        
        futureMaxEnd = min(index+self.maxValueLookahead, len(allStates))
        futureMaxes = map(lambda x: x["bars"][key]["max"], allStates[index+1:futureMaxEnd])

        # This algorithm trusts that previous values are correct. If there's an
        # error with the first value, then we need something to replace it with.
        # Look for the most frequent value in the next bunch of values
        if index == 0:
            histogram = {}
            maxOccurrences = 0
            mostFrequent = None
            for i in futureMaxes:
                s = str(i)
                if s in histogram:
                    histogram[s] = histogram[s] + 1
                else:
                    histogram[s] = 1
                if histogram[s] > maxOccurrences:
                    mostFrequent = i
                    maxOccurrences = histogram[s]

            if self.debug:
                print "Using {0} for first {1} value".format(mostFrequent, key)

            prevMax = mostFrequent
        else:
            prevMax = allStates[index-1]["bars"][key]["max"]

        # The health and mana ocr will output a -1 if the result is not a valid integer.
        # We know that's an ocr error, so replace it with the previous value
        if curMax == -1:
            if self.debug:
                print "Explicit OCR error in {0} at frame {1}: Replacing with {2}".format(
                    key, allStates[index]["frame_num"], prevMax)
            needsFix = True

        # Detect very large changes
        elif abs(curMax - prevMax) > self.maxChangeThreshold:
            if self.debug:
                print "Max change by too much. {0} -> {1}".format(prevMax, curMax)
            needsFix = True

        # If max health/mana changes, it should be consistent in the near future, meaning it shouldn't bounce back
        # If it goes up, the majority of the next states should be at or higher than that new value
        # If it goes down, the majority of the next states should be at or lower than that new value
        elif curMax != prevMax:
            if curMax > prevMax:
                consistentFutureMaxes = filter(lambda x: x-curMax >= 0, futureMaxes)
            else:
                consistentFutureMaxes = filter(lambda x: curMax-x >= 0, futureMaxes)
            
            if len(consistentFutureMaxes) < self.maxValueLookahead / 2:
                if self.debug:
                    print "Max jitter error in {0} at frame {1}: Found {2}, but saw {3} before and only {4} similar values after".format(
                               key, allStates[index]["frame_num"], curMax, prevMax, len(consistentFutureMaxes))
                needsFix = True

        if needsFix:
            cur["max"] = prevMax
            return True
        return False

    def fixCur(self, allStates, index, key):
        cur = allStates[index]["bars"][key]
        curVal = cur["current"]
        # What the value should be based on the percent value
        expectedCur = cur["percent"] * cur["max"] / 100
        
        prev = None
        if index > 0:
            prev = allStates[index-1]["bars"][key]["current"]

        if abs((expectedCur - curVal)/float(cur["max"])) > self.percentageThreshold:
            if self.debug:
                print "Percentage violation in {0} at frame {1}: Expected {2}, saw {3}. Error = {4}".format(
                    key, allStates[index]["frame_num"], expectedCur, curVal, abs((expectedCur - curVal)/float(cur["max"])))

            # We know that the expectedCur is not the exact value because it has a small error
            # We should use the previous value if it's within the threshold range
            if prev != None and abs((expectedCur - prev)/float(cur["max"])) <= self.percentageThreshold:
                cur["current"] = prev
                return True
            else:
                cur["current"] = expectedCur
                return True
        return False

