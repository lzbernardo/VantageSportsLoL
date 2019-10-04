import argparse, os

import cv2
import numpy as np

import spatial_layout
from img_util import percent_in_range
from util import imprev

class HealthManaCalc:
    def __init__(self, layout, debug=False):
        self.layout = layout
        self.debug = debug
        basedir = os.path.dirname(os.path.realpath(__file__)) + "/"

        self.model = cv2.KNearest()
        trainingdir = basedir + 'training_data/'
        samples = np.loadtxt(trainingdir + 'h_samp_1280.data', np.float32)
        responses = np.loadtxt(trainingdir + 'h_resp_1280.data', np.float32)
        self.model.train(samples, responses)

        templatedir = basedir + 'Templates/'
        self.slash_temps = [cv2.imread(templatedir + 'h_slash_1280_%s.png' % i) for i in range(1,5)]

    def get_health_and_mana(self, frame):
        # Try to read the health numbers
        health_text = self.extract_text(frame, self.layout.champ_detail.health_text)
        mana_text = self.extract_text(frame, self.layout.champ_detail.mana_text)

        # Get the percent of pixels in the champ_detail's health bar
        # that aren't black. this value is expected with be within +/- 2% of the true
        # (text-stated) value.
        # look at just a small slice of the bar that isn't ever obscured by text
        health_slice_img = self.layout.champ_detail.health.img(frame)[1:3,1:-1]
        health_percent = percent_in_range(health_slice_img, (0, 50, 0), (255, 255, 255))

        # Get the percent of pixels in the champ_detail's mana bar
        # that aren't black. this value is expected with be within +/- 2% of the true
        # (text-stated) value.
        # look at just a small slice of the bar that isn't ever obscured by text
        mana_slice_img = self.layout.champ_detail.mana.img(frame)[1:3, 1:-1]
        mana_percent = percent_in_range(mana_slice_img, (50, 0, 0), (255, 255, 255))
        
        return {
            'health': {
                'current': health_text[0],
                'max': health_text[1],
                'percent': int(round(health_percent)),
            },
            'mana': {
                'current': mana_text[0],
                'max': mana_text[1],
                'percent': int(round(mana_percent)),
            },
        }

    def extract_text(self, frame, text_bar):
        char_width = self.layout.champ_detail.bar_char_width
        slash_gap = self.layout.champ_detail.bar_slash_gap
        slash_width = self.layout.champ_detail.bar_slash_width

        bar_img = text_bar.img(frame)
        slash_offset = self.slashfind(bar_img)

        left_text_offset = slash_offset - slash_gap - (4 * char_width)
        right_text_offset = slash_offset + slash_width + slash_gap

        left_text = self.knn_string_build(left_text_offset, bar_img)
        right_text = self.knn_string_build(right_text_offset, bar_img)

        return [ left_text, right_text ]

    # Getting location of slash, in number of pixels from left of frame cropped
    # to health/mana bar; allow_sub includes cases for imperfect matching
    def slashfind(self, bar, slash_match_threshold=0.69, allow_sub=True):
        slash_window = self.layout.champ_detail.bar_slash_window
        bar = bar[:, slash_window[0]:slash_window[1]]

        slash_matches = [ cv2.matchTemplate(bar, slash_temp, cv2.TM_CCOEFF_NORMED)[0] for slash_temp in self.slash_temps ]
        best_matches = np.maximum.reduce(slash_matches)
        slash_loc = np.where(best_matches == best_matches.max())[0]
        
        if len(slash_loc) > 1:
            # There was a tie for first. Use the first value.
            if self.debug:
                print 'found %d slashes' % len(slash_loc)
            slash_loc = [slash_loc[0]]

        if best_matches[slash_loc] < slash_match_threshold:
            if self.debug:
                print 'best match was below threshold:', best_matches[slash_loc]

        if self.debug:
            print 'slash_loc', slash_loc
            imprev(bar)

        return slash_loc[0] + slash_window[0]

    def knn_string_build(self, digit_pos, bar, red_threshold=80):
        knn_string = ''
        char_width = self.layout.champ_detail.bar_char_width
        for i in range(4):
            start =  max(0, digit_pos + (i * char_width))
            end = min(start + char_width, len(bar[0]))
            # Examine the digit area for red pixels greater than the threshold
            # (indication of white text, since backgrounds are blue and green).
            digit_subimg = bar[:, start:end]
            thresh = cv2.inRange(digit_subimg, (0, 0, red_threshold), (255, 255, 255))
            num_red = cv2.countNonZero(thresh)

            if num_red < 5:
                knn_char = ' '
            else:   
                knn_char = self.knn_match(digit_subimg)

            if self.debug:
                print knn_char
                imprev(digit_subimg)
            knn_string += knn_char

        knn_string = knn_string.strip()
        try:
            return int(knn_string)
        except ValueError:
            # TODO: We should use these as test cases to improve this algorithm
            return -1

    def flatten(self, img):
        return img.reshape((1,img.shape[0]*img.shape[1]))

    def knn_match(self, digit, k_val=1):
        # Convert the digit image into a float array of the flattened grayscale image
        flat_grayscale = self.flatten(cv2.cvtColor(digit, cv2.COLOR_BGR2GRAY))
        float_vals = np.float32(flat_grayscale)

        # Try to find the closest match in our model to that float array
        nearest_float = self.model.find_nearest(float_vals, k=k_val)[0]

        # We get an ascii value of the closest match. Return the actual character
        return chr(int(nearest_float))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Given a frame image, prints the health and mana values and percents')
    parser.add_argument('-i','--img', help='input image', required=True)
    parser.add_argument('-v','--verbose', action='store_true', help='verbose debugging info')
    args = parser.parse_args()

    frame = cv2.imread(args.img)
    layout = spatial_layout.layout_1280x1024_6_10()

    hAndm = HealthManaCalc(layout, debug=args.verbose).get_health_and_mana(frame)
    print "Health", hAndm['health']
    print "Mana", hAndm['mana']

    imprev(frame, label=args.img, mag=False, scale=1)

