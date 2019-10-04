# LOL

This repo contains the API models and methods for fetching data from the offical riot developer APIs. 

[Official Documentation](developer.riotgames.com)

### ID Distribution

I ran a quick experiment to find out what the NA1 ID distribution looked like...
```
./fetch_match_abbrs --api_key $API_KEY --out /tmp/matches --errors /tmp/errors --start_id 1300000000 --end_id 1989413200 --per_ten 3000 --workers 800 --incr 10000

grep -o -P '"match_id".{0,11}|"date".{0,15}' /tmp/matches | paste -d " "  - - | sort -n -k 2 | grep "T00" | uniq -s 25 | cut --complement -c 1-11 | cut --complement -c 12-19
```

* 1365540000	2014-05-01T00
* 1403160000	2014-06-01T00 (~31m)
* 1438560000	2014-07-01T00 (~31m)
* 1476420000	2014-08-01T00 (~32m)
* 1517620000	2014-09-01T00 (~40m)
* 1564540000	2014-10-01T00 (~42m)
* 1614510000	2014-11-01T00 (~42m)
* 1652270000	2014-12-02T00 (~32m)
* 1683660000	2015-01-01T00 (~30m)
* 1716310000	2015-02-01T00 (~30m)
* 1746620000	2015-03-01T00 (~30m)
* 1778990000	2015-04-01T00 (~30m)
* 1810290000	2015-05-01T00 (~30m)
* 1842730000	2015-06-01T00 (~30m)
* 1872940000	2015-07-01T00 (~30m)
* 1905450000	2015-08-01T00 (~30m)
* 1937370000	2015-09-01T00 (~30m)
* 1966080000	2015-10-01T00 (~22m)

### Fetching Matches

Note: depending on the parallelism you're after, you may need to increase the maximum number of open files (for all the sockets that will be opened on your behalf). If you're going to run with several hundred workers, setting it to 3000 should be more than enough.
```
ulimit -n
# Increase it if necessary
ulimit -n 3000
```

To fetch the batch of 50000 sorta recent matches (1969400000 - 1969450000) with 800 "threads" run the following.
```
cd $GOPATH/src/github.com/VantageSports/lol/cmd/fetch_match_abbrs

go build

time ./fetch_match_abbrs --api_key $API_KEY --out /tmp/matches \
	--errors /tmp/errors --start_id 1969400000 --end_id 196950000 \
	--per_ten 3000 --workers 800 --verbose 2> /tmp/fetch.log
```

### Understanding the data

If you redirected stderr to the /tmp/fetch.log file (as in the example above) you can use that file to understand how your fetch is performing.

Total number of requests made:
```
grep "REQ" /tmp/fetch.log | wc -l
```

Total number of retries made:
```
grep "RETRY" /tmp/fetch.log | wc -l
```

Total number of requests made during each 10-second increment:
```
grep "REQ" /tmp/fetch.log | cut -c1-18 | sort | uniq -c
```

Total number of requests made during each 10-minute increment:
```
grep "REQ" log.* | cut -d: -f2-3 | sed 's/.$//' | sort -g | uniq -c
```

Total number of retries made during each of the 10-second increments:
```
grep "RETRY" /tmp/fetch.log | cut -c1-18 | sort | uniq -c
```


### Putting data into Big Query

Lets say that you wanted to put 100 matches into a new big query table. This is how you would do it.

First, get 100 matches (assuming you're using a dev API key, so 10 per sec)

```
./fetch_match_abbrs --api_key $API_KEY --out /tmp/matches \
	--errors /tmp/errors --start_id 1969400001 --end_id 1969400100 \
	--per_ten 8 --workers 8 --verbose 2> /tmp/fetch.log
```

This should take about 2 minutes to run at this rate. 

Then we'll load it into a table (here tmp1017):

```
bq load --schema ../../vs_model.bq.json --source_format NEWLINE_DELIMITED_JSON lol.tmp1017 /tmp/matches
```

This may take up to another minute or two, but you can test that it worked by running a query against it.

```
bq query 'select count(*) from lol.tmp1017'
```

### How to generate 10m match summaries:

As long as nobody else is using your same API KEY, you should be able to max out the rate-limit of a production key from a single machine. I recommend using somewhere between 500 and 1000 workers, and make sure your ulimit is set high enough (4000?).
