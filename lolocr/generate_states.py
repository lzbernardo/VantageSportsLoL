import argparse, json, sys, os.path

import cv2

from video_props import VideoProps
from spatial_layout import find_layout
from frame_state import states_from_video, state_from_frame

sys.path.append(os.path.join(os.path.dirname(__file__), 'correction'))
from post_process.correct_states import correct_states

def main():
    parser = argparse.ArgumentParser(description='LolOCR python wrapper. Generates raw ocr states from input video.')
    parser.add_argument('-i', '--video', help='video path', required=True)
    parser.add_argument('-v', '--version', help='match version', default='6.10')
    parser.add_argument('-o', '--out', help='output file', required=True)
    parser.add_argument('-s', '--start', help='start frame', default=0, type=int)
    parser.add_argument('-e', '--end', help='end frame', default=-1, type=int)
    parser.add_argument('-f', '--fps', help='frames per second to analyze', default=10, type=int)
    parser.add_argument('--pretty', help='if true, indents the output', dest='pretty', action='store_true')
    parser.add_argument('-p','--participant', help='participant id', type=int, required=True)
    args = parser.parse_args()

    # initialization / setup
    cap = cv2.VideoCapture(args.video)
    props = VideoProps(cap)
    layout = find_layout(args.version, props.width(), props.height())
    cap.release()

    if args.end < 0:
        args.end = props.frames()

    if args.end <= args.start:
        print 'end frame must be before start frame'
        sys.exit(1)

    print 'state tracking %s %s' % (args.video, props.__dict__)
    states = states_from_video(
        video_path=args.video,
        props=props,
        layout=layout,
        first_frame=args.start,
        last_frame=args.end,
        frames_per_sec=args.fps,
        participant=args.participant)

    f = open(args.out, 'w')
    if args.pretty:
        j = json.dumps(states, indent=4)
    else:
        j = json.dumps(states)
    f.write(j)
    f.close()


if __name__ == '__main__':
    main()
