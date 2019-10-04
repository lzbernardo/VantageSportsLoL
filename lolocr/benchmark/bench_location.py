# To record results:
# $ python bench_location.py > location/results.txt 2>&1

import argparse, os.path, sys, json, glob

import cv2

sys.path.append('..')
import spatial_layout
from frame_state import champ_location
import util

# These sample points were gathered from actual riot .json files and represent locations
# near the four corners of the map.
points = [[222, 471], [351, 293], [14486, 14511], [14237, 14579], [12860, 1670], [2235, 13315], [13032, 1928], [2256, 12897], [396, 263], [12876, 1701], [1217, 12381], [12706, 1658], [1307, 12262], [14363, 14726], [12408, 1441], [1642, 12834], [13250, 2228], [1840, 12608], [12720, 1688], [1868, 13182], [12816, 1614], [1819, 13142], [12489, 1557], [2038, 13193], [13486, 2692], [1927, 13081], [12484, 1550], [1852, 12992], [13159, 2098], [1140, 12229], [12725, 1571], [1145, 12248], [13300, 2256], [2238, 13372], [14589, 14454], [13238, 2307], [1150, 12086], [12518, 1531], [1073, 12209], [13372, 2494], [2152, 12890], [12696, 1790], [1680, 12598], [13300, 2280], [2259, 13388], [13118, 1964], [1852, 12842], [12639, 1550], [1714, 12834], [12739, 1719], [1609, 12962], [662, 285], [12767, 1731], [1240, 12083], [12493, 1357], [2198, 13505], [13283, 2244], [1304, 12352], [12483, 1459], [1727, 12985], [12287, 1212], [1686, 12955], [12369, 1356], [1461, 12705], [12274, 1243], [2341, 13450]]

def main(x_offset=0, y_offset=0, quiet=False):
    parser = argparse.ArgumentParser(description='Tests location ocr')
    parser.add_argument('-v','--verbose', action='store_true', help='verbose debugging info')
    args = parser.parse_args()
    layout = spatial_layout.layout_1280x1024_6_10()
    if x_offset != 0 and y_offset != 0:
        layout.minimap.x_offset = x_offset
        layout.minimap.y_offset = y_offset

    expectations = None
    with open("location/expectations.json") as exFile:
        expectations = json.load(exFile)

    total_successes = 0
    total_success_errors = 0
    success_threshold_percent = 5
    test_img_files = glob.glob("location/*.jpg")
    test_img_files.sort()
    for img_file in test_img_files:
        frame = cv2.imread(img_file)

        location = champ_location(frame, layout)

        errors = match(location, expectations[os.path.basename(img_file)], debug=args.verbose)
        # If all the error values are less than the threshold, then it's a success
        if len(filter(lambda x:x < success_threshold_percent, errors)) == len(errors):
            total_successes += 1
            total_success_errors += errors[0] + errors[1]

        if not quiet:
            print "{0} - error = {1:.2f}%, {2:.2f}%".format(img_file, errors[0], errors[1])

    print "{0}/{1} are within {2}% error".format(total_successes, len(test_img_files), success_threshold_percent)
    return total_success_errors/float(2*total_successes)

def match(location, expected, debug=False):
    if debug:
        print "\tSaw location: {0},{1}, Expected location: {2},{3}".format(
            location["x"], location["y"],
            expected["x"], expected["y"])

    # The map bounds are x: [0,14820] and y: [0:14881]
    # See https://developer.riotgames.com/docs/game-constants
    return [abs(location["x"]-expected["x"])/float(148.2),
            abs(location["y"]-expected["y"])/float(148.81)]

def draw_point(frame, layout, location, pixel=[0,255,0]):
    map_box = layout.minimap.box
    x = location['x'] * (map_box.width()/float(14820))
    y = location['y'] * (map_box.height()/float(14881))
    xyBox = spatial_layout.BoundingBox(map_box.bottom - y,
                                       map_box.bottom - y + 1,
                                       map_box.left + x,
                                       map_box.left + x + 1)

    util.draw_bounding_box(frame, xyBox, pixel=pixel)

def draw_points(frame, layout, points, pixel=[0,255,0]):
    for point in points:
        draw_point(frame, layout, {"x":point[0], "y":point[1]}, pixel)


if __name__ == '__main__':
    findMinErrorMode = False

    if findMinErrorMode:
        # Adjust these ranges to find a combination that yields a low average error
        for x in range(122,123):
            for y in range(-187,-186):
                error = main(x_offset=x, y_offset=y, quiet=True)
                print "Average error of successes: {0}, {1} = {2:.6f}%".format(x,y,error)

    else:
        error = main()
        print "Average error of successes: {0:.6f}%".format(error)
