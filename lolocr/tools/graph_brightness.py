import argparse, json, sys

def main():
    parser = argparse.ArgumentParser(description='graph brightness')
    parser.add_argument('-i','--input', help='json input', required=True)
    parser.add_argument('-o','--out', help='output file', required=True)
    args = parser.parse_args()

    with open(args.input) as in_file, open(args.out, 'w') as out_file:
        data = json.load(in_file)

        for frame in data:
            out_file.write(",".join((
                str(frame["frame_num"]), 
                str(frame["brightness"]["innate"]),
                str(frame["brightness"]["q"]),
                str(frame["brightness"]["w"]),
                str(frame["brightness"]["e"]),
                str(frame["brightness"]["r"]),
                str(frame["brightness"]["spell1"]),
                str(frame["brightness"]["spell2"]),
            )))
            out_file.write("\n")

if __name__ == '__main__':
    main()
