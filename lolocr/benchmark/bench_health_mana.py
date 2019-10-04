# To record results:
# $ python bench_health_mana.py > health_mana/results.txt 2>&1

import argparse, os.path, sys, json, glob

import cv2

sys.path.append('..')
import spatial_layout
from health_mana_calc import HealthManaCalc

def main():
    parser = argparse.ArgumentParser(description='Tests health/mana ocr')
    parser.add_argument('-v','--verbose', action='store_true', help='verbose debugging info')
    args = parser.parse_args()
    layout = spatial_layout.layout_1280x1024_6_10()

    expectations = None
    with open("health_mana/expectations.json") as exFile:
        expectations = json.load(exFile)

    total_successes = 0
    test_img_files = glob.glob("health_mana/*.jpg")
    test_img_files.sort()
    for img_file in test_img_files:
        frame = cv2.imread(img_file)

        hAndm = HealthManaCalc(layout).get_health_and_mana(frame)

        num_successes = match(hAndm, expectations[os.path.basename(img_file)], debug=args.verbose)
        print "{0} - {1}/4 correct".format(img_file, num_successes)

        total_successes += num_successes

    print "{0}/{1} succeeded - {2:.2f}%".format(total_successes, 4 * len(test_img_files), total_successes * 100 / float(4*len(test_img_files)))

def match(calculated, expected, debug=False):
    num_correct = 0
    if debug:
        print "\tSaw health: {0} / {1}, Expected health: {2} / {3}".format(
            calculated["health"]["current"],
            calculated["health"]["max"],
            expected["health"]["current"],
            expected["health"]["max"])
        print "\tSaw mana: {0} / {1}, Expected mana: {2} / {3}".format(
            calculated["mana"]["current"],
            calculated["mana"]["max"],
            expected["mana"]["current"],
            expected["mana"]["max"])

    # If there's a ? in the expected, then it doesn't matter what the ocr says
    if str(calculated["health"]["current"]) == expected["health"]["current"] or "?" in expected["health"]["current"]:
        num_correct += 1
    if str(calculated["health"]["max"]) == expected["health"]["max"] or "?" in expected["health"]["max"]:
        num_correct += 1
    if str(calculated["mana"]["current"]) == expected["mana"]["current"] or "?" in expected["mana"]["current"]:
        num_correct += 1
    if str(calculated["mana"]["max"]) == expected["mana"]["max"] or "?" in expected["mana"]["max"]:
        num_correct += 1

    return num_correct

if __name__ == '__main__':
    main()
