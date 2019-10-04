# spacial_layout describes the on-screen layout given a certain patch number
# and screen resolution. To keep things simple, we try to keep videos at
# 1280w x 1024h, but this config allows us to change that out.

import os

import numpy as np

basedir = os.path.dirname(os.path.realpath(__file__))

# BoundingBox represents a 4-point section of a frame.
class BoundingBox(object):

    def __init__(self, top, bottom, left, right):
        self.top = top
        self.bottom = bottom
        self.left = left
        self.right = right

    def img(self, frame):
        return frame[self.top:self.bottom, self.left:self.right]

    def height(self):
        return self.bottom - self.top

    def width(self):
        return self.right - self.left

    def offset(self, off_top, off_bottom, off_left, off_right):
        return BoundingBox(self.top + off_top, self.bottom + off_bottom, self.left + off_left, self.right + off_right)

# Layout is the patch-specific interface for tracking where key pieces of visual
# state are located.
class Layout(object):

    def __init__(self, templates, champ_detail, stats_table, minimap):
        self.templates = templates
        self.champ_detail = champ_detail
        self.stats_table = stats_table
        self.minimap = minimap

# Templates is an object with functions that return images to be used for
# template matching.
class Templates(object):

    def __init__(self, dir):
        self.dir = dir

    # This function generates an image that looks just like the top-right corner
    # of the minimap camera box. It can be rotated to match other corners.
    def camera_box_corner_top_right(self):
        corner = np.zeros((16, 16, 1), np.uint8)
        corner[:2,:14] = 255
        corner[2:,14:] = 255
        return corner


# ChampDetail describes the lower-left champ-detail UI elements.
class ChampDetail(object):

    def __init__(self, props):
        self.icon = props["icon"]
        self.health = props["health"]
        self.health_text = props["health_text"]
        self.mana = props["mana"]
        self.mana_text = props["mana_text"]
        self.experience = props["experience"]
        self.innate = props["innate"]
        self.q = props["q"]
        self.w = props["w"]
        self.e = props["e"]
        self.r = props["r"]
        self.spell1 = props["spell1"]
        self.spell2 = props["spell2"]
        # Offset within the health/mana bars to search for slash; helps with false positives
        self.bar_slash_window = props["bar_slash_window"]
        self.bar_slash_width = props["bar_slash_width"]
        # Margins to the left and right of the slash
        self.bar_slash_gap = props["slash_gap"]
        self.bar_char_width = props["bar_char_width"]

# StatsTable describes the lower-middle champ stats table.
class StatsTable(object):

    def __init__(self, all_icons, icon_height, scrub_bar):
        self.all_icons = all_icons
        self.icon_height = icon_height
        self.scrub_bar = scrub_bar

# MiniMap describes the lower-right miniature map.
class MiniMap(object):

    def __init__(self, box, camera_height, camera_width, x_offset, y_offset):
        self.box = box
        # These are estimates that are used when we can't confidently determine the
        # bounding box
        self.camera_height = camera_height
        self.camera_width = camera_width
        # The true location of the champion tends to be slightly offset
        # from the center of the camera box. See bench_location.py to help
        # determine these values
        self.x_offset = x_offset
        self.y_offset = y_offset

def layout_1280x1024_6_10():
    templates = Templates(basedir + '/templates/1280x1024')
    champ_detail = ChampDetail({
        "icon": BoundingBox(825, 875, 14, 64),
        "health": BoundingBox(825, 842, 71, 161),
        "health_text": BoundingBox(829, 837, 83, 149),
        "mana": BoundingBox(845, 862, 71, 161),
        "mana_text": BoundingBox(849, 857, 83, 149),
        "experience": BoundingBox(865, 871, 71, 161),
        "innate": BoundingBox(883, 913, 12, 42),
        "q": BoundingBox(883, 913, 45, 75),
        "w": BoundingBox(883, 913, 78, 108),
        "e": BoundingBox(883, 913, 111, 141),
        "r": BoundingBox(883, 913, 144, 174),
        "spell1": BoundingBox(883, 913, 183, 213),
        "spell2": BoundingBox(883, 913, 217, 247),
        "bar_slash_window": (21, 38),
        "bar_slash_width": 6,
        "slash_gap": 2,
        "bar_char_width": 5
    })
    stats_table=StatsTable(
        all_icons=BoundingBox(880, 1013, 612, 672),
        icon_height=23,
        scrub_bar=BoundingBox(830, 832, 530, 800))

    minimap = MiniMap(
        box=BoundingBox(745, 1009, 1001, 1268),
        camera_height=42,
        camera_width=53,
        x_offset=122,
        y_offset=-187)

    return Layout(templates, champ_detail, stats_table, minimap)

# find_layout returns the correct layout based on the path
def find_layout(patch_num=6.10, width=1280, height=1024):
    if width == 1280 and height == 1024:
        return layout_1280x1024_6_10()
    return None
