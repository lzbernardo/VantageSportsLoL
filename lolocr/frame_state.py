# frame_state detects state from video image frames.

import ast, json, math, os, time
from collections import Counter

# Note: As of June '16, we've pinned our OpenCV libraries to 2.4.12.
import cv2
import numpy as np
import health_mana_calc

from img_util import brightness, rotate, percent_in_range
from spatial_layout import BoundingBox

# states_from_video iterates through every frame in the video between
# first_frame and last_frame (inclusive), calling state_from_frame on each frame
# that shouldn't be skipped eventually returning all non-nil states in an array
# to the caller.
def states_from_video(video_path, props, layout, participant, first_frame, last_frame, frames_per_sec):
    frame_num = 0
    skip_mod = int(math.ceil(props.fps() / frames_per_sec))
    states = []

    hmCalc = health_mana_calc.HealthManaCalc(layout)
    # we don't actually start examining frames until it appears that the HUD
    # is in place (which contains most of the details we care about)
    game_started = False

    cap = cv2.VideoCapture(video_path)
    while True:
        frame_num += 1
        if frame_num % 1000 == 0:
            print 'frame %d / %d' % (frame_num, last_frame)
        if frame_num > last_frame:
            break
        if frame_num < first_frame or (frame_num % skip_mod != 0):
            cap.grab()
            continue

        ret, frame = cap.read()
        if not ret:
            break

        # We don't care about the portion of the video shown before the HUD
        # is displayed.
        game_started = game_started or is_scrub_bar_shown(frame, layout)
        if not game_started:
            continue

        state = state_from_frame(frame, layout, participant, hmCalc)
        if state:
            state['frame_num'] = frame_num
            state['video_sec'] = frame_num / props.fps()
            state['video_clock'] = '%02d:%02d' % divmod(state['video_sec'], 60)
            states.append(state)

    cap.release()
    return states

# state_from_frame returns the observed state of a single frame.
def state_from_frame(frame, layout, participant, hmCalc):
    state = {}
    state['pov_champ'] = pov_champ_num(frame, layout)

    if participant != state['pov_champ']:
        # TODO(Cameron): consider continuing here? We could use it to
        # corroborate the data in other videos perhaps, though we'd need to be
        # careful to scrub non-desired-champ frames in post-processing.
        return None

    state['location'] = champ_location(frame, layout)
    if not state['location']:
        return state

    state['brightness'] = champ_ability_spell_readyness(frame, layout)
    healthAndMana = hmCalc.get_health_and_mana(frame)
    state['bars'] = {
        'health': healthAndMana['health'],
        'mana': healthAndMana['mana'],
        'experience': champ_detail_experience(frame, layout)
    }

    return state


# The spectator scrub bar appears to be the least transparent and darkest black
# area of the spectator HUD, so we look for the presense of a mostly black area
# there to determine if the HUD is shown, which we can use as a proxy for
# "start of game".
def is_scrub_bar_shown(frame, layout):
    scrub_bar_img = layout.stats_table.scrub_bar.img(frame)
    hsv = cv2.cvtColor(scrub_bar_img, cv2.COLOR_BGR2HSV)

    # This is the greenish/blue color of the scrub bar to the left of the
    # position indicator.
    played_pct = percent_in_range(hsv, (75, 0, 68), (82, 255, 75))
    # This is the blackish color of the scrub bar to the right of the position
    # indicator.
    unplayed_pct = percent_in_range(hsv, (0, 0, 0), (180, 255, 5))

    return (unplayed_pct + played_pct) > 90


# pov_champ_num returns the participant number (1 - 10) of the participant
# with camera-lock on, or -1 if not certain which champ is camera-locked.
def pov_champ_num(frame, layout):
    icon = layout.champ_detail.icon
    all_icons = layout.stats_table.all_icons

    icon_img = icon.img(frame)
    all_img = all_icons.img(frame)

    # scale the image down to the size of a stats_table icon
    scale = 1.0 * layout.stats_table.icon_height / icon.height()
    scaled_icon = cv2.resize(icon_img, None, fx=scale, fy=scale)

    result = cv2.matchTemplate(all_img, scaled_icon, cv2.TM_CCORR_NORMED)
    (_, max_val, _, loc) = cv2.minMaxLoc(result)

    # did we find a reasonable match?
    if max_val < 0.75:
        return -1

    # great - now find which of the 10 possible locations it is in.
    (x, y) = loc
    pos = int(round(1.0 * y / all_icons.height() * 5))
    team = int(round(1.0 * x / all_icons.width() * 2))
    return 1 + pos + team * 5


# camera_box finds the minimap "box" showing what portion of the map the game
# camera is looking at. It does so by constructing a simple corner
# constructs a simple "corner" image itself, and then rotates it until it finds
# a good match, interpolating the camera box based on the corner orientation and
# image size.
#
# NOTE: This bounding box's dimensions are defined relative to the minimap's
# box, not the overall frame.
def camera_box(frame, layout):
    # convert the minimap to a thresholded grayscale image
    gray = cv2.cvtColor(layout.minimap.box.img(frame), cv2.COLOR_BGR2GRAY)
    _, thresh_map = cv2.threshold(gray, 230, 255, cv2.THRESH_BINARY)

    corner = layout.templates.camera_box_corner_top_right()

    # TODO(Find programatically)
    cam_height = layout.minimap.camera_height
    cam_width = layout.minimap.camera_width
    cam_h_offset = -1 * (43 - len(corner))
    cam_w_offset = -1 * (55 - len(corner))
    # depending on how much the corner is rotated, we need to adjust the
    # location of the camera box.
    xy_offsets = [
        (cam_w_offset, 0),            # top-right
        (cam_w_offset, cam_h_offset), # bottom-right corner
        (0, cam_h_offset),            # botom-left corner
        (0, 0)                        # top-left
    ]

    for i in range(4):
        result = cv2.matchTemplate(thresh_map, corner, cv2.TM_CCORR_NORMED)
        (_, max_val, _, loc) = cv2.minMaxLoc(result)
        if max_val > 0.80:
            left = loc[0] + xy_offsets[i][0]
            top = loc[1] + xy_offsets[i][1]

            return BoundingBox(top, top + cam_height, left, left + cam_width)
        # otherwise, rotate and try again.
        corner = rotate(corner, -90) # clockwise

# champ_location infers the location of the camera-locked summoner by finding
# the presumed middle of the camera_box, and then calculating the x,y offsets
# from the bottom left corner of the minimap and multiplying by the riot grid
# max, which is currently 14820 x 14881
def champ_location(frame, layout):
    cam_box = camera_box(frame, layout)
    max_x = 14820
    max_y = 14881
    if not cam_box:
        return {'x': -1, 'y': -1}

    map_box = layout.minimap.box
    x = cam_box.left + cam_box.width() / 2
    pos_x = max_x * x / map_box.width() + layout.minimap.x_offset
    y = map_box.height() - (cam_box.top + cam_box.height() / 2)
    pos_y = max_y * y / map_box.height() + layout.minimap.y_offset

    return {
        'x': min(max(pos_x, 0), max_x),
        'y': min(max(pos_y, 0), max_y),
    }

# champ_ability_spell_readyness returns the 'brightness' of each of the ability
# and spell icons. this data can be used to infer which abilities are on
# cooldown and which are ready to be used based on relative brightness.
def champ_ability_spell_readyness(frame, layout):
    return {
        'innate': brightness(layout.champ_detail.innate.img(frame)),
        'q': brightness(layout.champ_detail.q.img(frame)),
        'w': brightness(layout.champ_detail.w.img(frame)),
        'e': brightness(layout.champ_detail.e.img(frame)),
        'r': brightness(layout.champ_detail.r.img(frame)),
        'spell1': brightness(layout.champ_detail.spell1.img(frame)),
        'spell2': brightness(layout.champ_detail.spell2.img(frame)),
    }

# champ_detail_experience gets the percent of pixels in the champ_detail's
# experience bar that are not black.
def champ_detail_experience(frame, layout):
    img = layout.champ_detail.experience.img(frame)[:, 1:-1]
    percent = percent_in_range(img, (0, 00, 50), (255, 255, 255))
    return { 'percent': int(round(percent)) }
