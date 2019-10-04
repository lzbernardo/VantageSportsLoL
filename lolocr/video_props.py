# capture_util contains utility functions for determining properties of the 
# video file (or image captures from the video) and one-off template matching.

import cv2

class Props(object):
    def __init__(self, fps=0, width=0, height=0, frames=0, codec=''):
        self._fps = fps
        self._width = width
        self._height = height
        self._frames = frames
        self._codec = codec

    def fps(self):
        return self._fps

    def frames(self):
        return self._frames

    def width(self):
        return self._width

    def height(self):
        return self._height

    def codec(self):
        return self._codec

    def seconds(self):
        fps = self.fps()
        frames = self.frames()
        return frames / fps if fps > 0 and frames > 0 else 0

# video_props returns a dict of basic video attributes (fps, w, h).
def VideoProps(cap):
    return Props(
        fps    = cap.get(cv2.cv.CV_CAP_PROP_FPS),
        frames = cap.get(cv2.cv.CV_CAP_PROP_FRAME_COUNT),
        width  = cap.get(cv2.cv.CV_CAP_PROP_FRAME_WIDTH),
        height = cap.get(cv2.cv.CV_CAP_PROP_FRAME_HEIGHT),
        codec  = cap.get(cv2.cv.CV_CAP_PROP_FOURCC),
    )
