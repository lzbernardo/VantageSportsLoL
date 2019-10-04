import argparse
import cv2

def main():
    parser = argparse.ArgumentParser(description='Video frame extractor')
    parser.add_argument('-i','--video', help='video path', required=True)
    parser.add_argument('-f', '--frames', help='frame number', required=True)
    args = parser.parse_args()

    framesToCap = args.frames.split(",")

    # initialization / setup
    cap = cv2.VideoCapture(args.video)
    frame_num = 0
    while True:
        frame_num += 1
        if frame_num % 1000 == 0:
            print 'frame %d' % (frame_num)
        if frame_num > int(framesToCap[-1]):
            break
        if str(frame_num) not in framesToCap:
            cap.grab()
            continue

        ret, frame = cap.read()
        cv2.imwrite("./{0}.jpg".format(frame_num), frame)

    cap.release()

if __name__ == '__main__':
    main()
