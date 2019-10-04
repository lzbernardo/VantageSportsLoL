import json
import cv2

# read_json reads the json object at the specified file path.
def read_json(path):
    f = open(path)
    j = json.loads(f.read())
    f.close()
    return j

# write_json serializes the specified object (with indentation if pretty==True)
# and writes the json object to the specified path.
def write_json(path, obj, pretty=False):
    f = open(path, 'w')
    str = json.dumps(obj) if not pretty else json.dumps(obj, indent=4)
    f.write(str)
    f.close()

# draw_bounding_box is useful for debugging. It overlays the bounding box on the
# source image
def draw_bounding_box(source, bounding_box, pixel=128):
    # Top line
    if bounding_box.top >= 0:
        source[bounding_box.top,max(bounding_box.left,0):bounding_box.right+1] = pixel
    # Right line
    if bounding_box.right < len(source[0]):
        source[max(bounding_box.top,0):bounding_box.bottom+1,bounding_box.right] = pixel
    # Bottom line
    if bounding_box.bottom < len(source):
        source[bounding_box.bottom,max(bounding_box.left,0):bounding_box.right+1] = pixel
    # Left line
    if bounding_box.left >= 0:
        source[max(bounding_box.top,0):bounding_box.bottom+1,bounding_box.left] = pixel

    return source

# imprev displays the image in a window
def imprev(img, label='img', mag=True, scale=10, interpolation=cv2.INTER_NEAREST):
    if mag is True:
        img = cv2.resize(img, (img.shape[1]*scale, img.shape[0]*scale), interpolation=interpolation)
    cv2.imshow(label, img)
    cv2.moveWindow(label, 0, 0)
    cv2.waitKey(0)
    cv2.destroyAllWindows()
