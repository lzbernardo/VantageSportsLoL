import sys, os

# This generates a bunch of csv files of before and after, that show the impact of the
# correction scripts.
# The input should be the uncorrected raw json that's output from generate_states.py
if __name__ == '__main__':
    videoJson = sys.argv[1]

    os.system("python correction/correct_states.py -i {0} -o {1}".format(videoJson, "clean_{0}".format(videoJson)))
    os.system("python tools/graph_health.py -i {0} -o {1}".format(videoJson, "{0}_health.csv".format(videoJson)))
    os.system("python tools/graph_health.py -i {0} -o {1}".format("clean_{0}".format(videoJson), "clean_{0}_health.csv".format(videoJson)))

    os.system("python tools/graph_brightness.py -i {0} -o {1}".format(videoJson, "{0}_brightness.csv".format(videoJson)))
    os.system("python tools/graph_brightness.py -i {0} -o {1}".format("clean_{0}".format(videoJson), "clean_{0}_brightness.csv".format(videoJson)))
    
    os.system("python tools/graph_location.py -i {0} -o {1}".format(videoJson, "{0}_location.csv".format(videoJson)))
    os.system("python tools/graph_location.py -i {0} -o {1}".format("clean_{0}".format(videoJson), "clean_{0}_location.csv".format(videoJson)))

              
