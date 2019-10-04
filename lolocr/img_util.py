import cv2
import numpy as np

# brightness returns the mean V value in the HSV spectrum of the image.
def brightness(img):
    hsv = cv2.cvtColor(img, cv2.COLOR_BGR2HSV)
    return int(cv2.mean(hsv[:,:,2])[0])

# rotates an image around its center by the specified number of degrees. note:
# positive angles are counter-clockwise.
def rotate(img, degrees):
    (h, w) = img.shape[:2]
    center = (w/2, h/2)
    matrix = cv2.getRotationMatrix2D(center, degrees, 1.0)
    return cv2.warpAffine(img, matrix, (w, h))

# percent_in_range returns the percentage of pixels in the specified image
# between the low and high values.
def percent_in_range(img, low, high):
    threshold = cv2.inRange(img, low, high)
    return 100.0 * cv2.countNonZero(threshold) / threshold.size
