import argparse, json, sys

def main():
    parser = argparse.ArgumentParser(description='graph location')
    parser.add_argument('-i','--input', help='json input', required=True)
    parser.add_argument('-o','--out', help='output file', required=True)
    args = parser.parse_args()

    with open(args.input) as in_file, open(args.out, 'w') as out_file:
        data = json.load(in_file)

        for frame in data:
            out_file.write(",".join((
                str(frame["frame_num"]),
                str(frame["location"]["x"]),
                str(frame["location"]["y"])
            )))
            out_file.write("\n")
            
if __name__ == '__main__':
    main()
