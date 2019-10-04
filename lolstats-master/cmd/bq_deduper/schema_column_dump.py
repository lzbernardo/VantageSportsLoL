#!/usr/bin/python

import json
import sys

def getColumns(data):
    field_list = []
    for i in data:
            field_list.append(i['name'])
            
    return field_list

def main():
    if len(sys.argv) != 2:
        print "Usage: {0} schema.json".format(sys.argv[0])
        return

    with open(sys.argv[1]) as data_file:
        data = json.load(data_file)

        s = getColumns(data)
        print ",".join(s)

if __name__ == "__main__":
    main()
