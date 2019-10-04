import argparse, json, sys

# This script takes the output of generate_states (the json file),
# and outputs a csv of frame,health,health_max that can be imported into a spreadsheet to graph
def main():
    parser = argparse.ArgumentParser(description='graph health')
    parser.add_argument('-i','--input', help='json input', required=True)
    parser.add_argument('-o','--out', help='output file', required=True)
    args = parser.parse_args()

    with open(args.input) as in_file, open(args.out, 'w') as out_file:
        data = json.load(in_file)

        for frame in data:
            out_file.write(",".join((
                str(frame["frame_num"]), 
                str(frame["bars"]["health"]["current"]),
                str(frame["bars"]["health"]["max"]))))
            out_file.write("\n")

if __name__ == '__main__':
    main()
