LOL PACKETS
=====

Utilities for decrypting and decoding the lol spectator block formats.


## Getting started decrypting blocks

Print the block summaries of all blocks
```
cd cmd/unpack
go build && ./unpack -in ../../replaytmp/2095022036/keyframe-02-dec.bin -summary=true
```

Print the contents of all blocks
```
go build && ./unpack -in ../../replaytmp/2095022036/keyframe-02-dec.bin -summary=true -contents=true
```

Documenting block types/formats
https://docs.google.com/a/vantagesports.com/spreadsheets/d/1A18LnD5lRU0gLBZmQlXNny-rYBdJozhfdOJpSCWmIJg/edit?usp=sharing
