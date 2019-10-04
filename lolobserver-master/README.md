# lolobserver

The "observer" is the process that notices when our customers are playing lol matches, and downloads (from a spectator server) the files necessary to watch that replay. Those replay files are persisted so that we can replay the match at some later time (to generate a video from).

There are two binaries that you'll find in the cmd directory.
 * observer: initiates a "user-watcher" that notices when a player enters a match. starts a download for each match discovered, and then writes a queue message to the match_details_ingest queue.
 * download_match: a one-off script for downloading a known match id with a known encryption key from a replay server (the riot spectator server, or a third-party like replay.gg)
