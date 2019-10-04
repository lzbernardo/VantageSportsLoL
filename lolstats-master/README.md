# LOLSTATS

[![wercker status](https://app.wercker.com/status/2bf5e9b346341e81cf5a30d0fb3e15bb/m/master "wercker status")](https://app.wercker.com/project/byKey/2bf5e9b346341e81cf5a30d0fb3e15bb)

There are two services in this repo

1. Match Stat Ingester
2. Stat Server

#Match Stat Ingester
* This service reads messages off the match_stat_ingest queue

#Stat Server

**Match History**
----

* **URL**

  /lolstats/v1/history
  
* **JSON Params**
  ```
  summoner_id: int, required
  platform: string, required
  limit: int, required

  purchase_state: string, optional
    If specified, then this will only return matches with a certain purchase state. Purchase states are defined in https://github.com/VantageSports/lolstats/blob/master/service/stat_row.go
  cursor: string, optional
    A string used for pagination. History responses will contain a cursor string. Passing that back in will return then next page of results
  ```

* **Example**
  ```
curl -H 'Authorization: Bearer <token>' -d '{"summoner_id":19887289, "platform":"NA1", "limit":1}' https://api-staging.vantagesports.com/lolstats/v1/history

{
  "matches": [
    {
      "created": "2016-05-06T18:33:24.202854Z",
      "last_updated": "2016-05-06T18:33:24.202854Z",
      "summonerId": 19887289,
      "platform": "NA1",
      "purchaseState": "UNAVAILABLE",
      "matchCreation": 1462298954148,
      "championId": 236,
      "lane": "BOTTOM",
      "championIndex": 8,
      "mapId": 11,
      "matchDuration": 2404,
      "matchId": 2177111321,
      "matchMode": "CLASSIC",
      "matchType": "MATCHED_GAME",
      "matchVersion": "6.8.141.4819",
      "offMeta": false,
      "opponentChampionId": 42,
      "queueType": "TEAM_BUILDER_DRAFT_RANKED_5x5",
      "role": "DUO_CARRY",
      "summonerName": "Imaqtpie",
      "winner": true,
      "tier": "CHALLENGER",
      "division": "I",
      "kills": 7,
      "deaths": 8,
      "assists": 14,
      "teamKills": 44,
      "enemyTeamKills": 39,
      "minionsKilled": 290,
      "csPerMinuteZeroToTen": 6.800000000000001,
      "csPerMinuteDiffZeroToTen": 0.9000000000000004,
      "neutralMinionsKilledPerMinute": 0.9234608985024958,
      "neutralMinionsKilledEnemyJunglePerMinute": 0.27454242928452577,
      "goldEarned": 18790,
      "goldZeroToTen": 262.29999999999995,
      "goldDiffZeroToTen": -6.800000000000068,
      "goldTenToTwenty": 519.2,
      "goldDiffTenToTwenty": 163.9000000000001,
      "goldTwentyToThirty": 627.7,
      "goldDiffTwentyToThirty": 236.50000000000006,
      "level6Seconds": 500.588,
      "level6DiffSeconds": 47.48499999999996,
      "wardsPlaced": 18,
      "wardsPlacedPerMinute": 0.4492512479201331,
      "totalDamageDealt": 264534,
      "totalDamageDealtPerMinute": 6602.346089850249,
      "totalDamageDealtToChampions": 26858,
      "totalDamageDealtToChampionsPerMinute": 670.332778702163,
      "totalDamageTaken": 36726,
      "totalDamageTakenPerMinute": 916.6222961730449,
      "totalHeal": 13949,
      "totalHealPerMinute": 348.1447587354409
    }
  ],
  "cursor": "ClcKGAoNbWF0Y2hDcmVhdGlvbhIHCKSTvb7HKhI3ag1zfnZzLXN0YWdpbmcxciYLEgpMb2xTdGF0Um93IhYyMTc3MTExMzIxLW5hLTE5ODg3Mjg5DBgAIAE"
}
```

**Means**
----

* **URL**

  /lolstats/v1/means
  
* **JSON Params**
  ```
  selects: []string, required
      A list of column names that you want to find the means for.
      Column names are the "datastore" value in https://github.com/VantageSports/lolstats/blob/master/service/stat_row.go
      There are also two special values, "kda", and "killParticipation". These get translated by the service
  
  // The following are a set of filters you can use to get the value you want. You can mix and match any combination you want.
  // At least one is required
  platform             : string, optional
      The platform in which the match was played. Ex. "NA1", "EUW1", etc.
  patch_prefix         : string, optional
      Limit results by a specific patch prefix. Ex. "6.7." matches
  tier                 : string, optional
      The solo queue tier to limit to. Ex. "DIAMOND", "CHALLENGER"
  division             : string, optional
      The division to limit to. Ex. "I", "II", "III", "IV", "V"
  lane                 : string, optional
      The lane the champion was played in. Ex. "TOP", "MIDDLE", "JUNGLE", "BOTTOM"
  role                 : string, optional
      The role the champion played in the game. Ex. "SOLO", "NONE", "DUO_SUPPORT", "DUO_CARRY"
      Most likely, you'll want to use the following combinations to match the meta:
        "top": {"lane": "TOP", "role": "SOLO"}
        "mid": {"lane": "MIDDLE", "role": "SOLO"}
        "jng": {"lane": "JUNGLE", "role": "NONE"}
        "sup": {"lane": "BOTTOM", "role": "DUO_SUPPORT"}
        "adc": {"lane": "BOTTOM", "role": "DUO_CARRY"}
  champion_id          : int, optional
      The champion_id that the summoner played. See https://developer.riotgames.com/docs/static-data and 
      http://ddragon.leagueoflegends.com/cdn/6.7.1/data/en_US/champion.json for champions in 6.7 patch.
      Parse the json, and look for the "key" value represents the champion_id
  opponent_champion_id : int, optional
      For games in which we can match up roles and lanes, this is the champion_id of the opponent opposite this summoner
  summoner_id          : int64, optional
      Limit results to just one particular summoner_id
  offMeta              : bool, optional
      Only count results that are on or off meta. If a game is off meta (value is true), then values of opponent_champion_id
      and stat diffs can be inaccurate
  ```

* **Returns**
  ```
  {
    "result": {
      "count": The number of records that were used to generate the averages
      <select_field_1>: <average for that field>
      <select_field_2>: <average for that field>
      ...
    }
  }
  ```

* **Example**
  Get the average kills, deaths, assists, kda ratio, and kill participation for Diamond mid laners in NA patch 6.7
  ```
  curl -H 'Authorization: Bearer <token>' -d '{"selects":["kills","deaths","assists", "kda", "killParticipation"],"platform":"NA1", "patch_prefix": "6.7.", "tier":"DIAMOND", "lane":"MIDDLE", "role":"SOLO", "offMeta":false}' https://api.vantagesports.com/lolstats/v1/means
{
  "result": {
    "assists": 7.428571428571429,
    "count": 49,
    "deaths": 5.285714285714286,
    "kda": 3.358079335630356,
    "killParticipation": 0.525039626842839,
    "kills": 5.877551020408164
  }
}
  ```

**Percentiles**
----

* **URL**

  /lolstats/v1/percentiles
  
* **JSON Params**
  Same as the means endpoint above

* **Returns**
  ```
    {
      "result": {
        "<select_key>": [
          <101 values. They represent the 0th to 100th percentile values for the select field passed in>
        ]
      }
    }
  ```
  
* **Example**
  The the percentile distribution of kda in NA Challenger mid laners in patch 6.7
  ```
  curl -H 'Authorization: Bearer <token>' -d '{"selects":["kda"],"platform":"NA1", "patch_prefix": "6.7.", "tier":"CHALLENGER", "lane":"MIDDLE", "role":"SOLO", "offMeta":false}' https://api.vantagesports.com/lolstats/v1/percentiles

{
  "result": {
    "kda": [
      0.2,
      0.2,
      0.3333333333333333,
      0.3333333333333333,
      0.5555555555555556,
      0.5555555555555556,
      0.75,
      0.75,
      0.8333333333333334,
      1,
      1,
      1.6,
      1.6,
      1.625,
      1.625,
      1.8,
      2,
      2,
      2,
      2,
      2.125,
      2.125,
      2.142857142857143,
      2.25,
      2.25,
      2.3333333333333335,
      2.3333333333333335,
      2.5,
      2.5,
      2.7142857142857144,
      2.7142857142857144,
      2.7142857142857144,
      2.75,
      2.75,
      2.75,
      2.75,
      2.857142857142857,
      3.25,
      3.25,
      3.3333333333333335,
      3.3333333333333335,
      3.3333333333333335,
      3.3333333333333335,
      3.6666666666666665,
      3.75,
      3.75,
      4,
      4,
      4,
      4,
      4.333333333333333,
      4.5,
      4.5,
      4.6,
      4.6,
      4.666666666666667,
      4.666666666666667,
      4.666666666666667,
      4.75,
      4.75,
      5,
      5,
      5,
      5,
      5,
      5,
      5,
      6,
      6,
      6,
      6,
      6.5,
      6.5,
      6.5,
      7,
      7,
      7.5,
      7.5,
      8,
      8,
      8,
      8.333333333333334,
      8.333333333333334,
      8.5,
      8.5,
      9,
      9.5,
      9.5,
      12,
      12,
      13,
      13,
      14,
      15,
      15,
      15,
      15,
      16,
      16,
      18,
      18
    ]
  }
}
```

**Match Details**
----

* **URL**

  /lolstats/v1/details
  
* **JSON Params**
  ```
  summoner_id: int, required
  platform: string, required
  match_id: int, required
  ```

* **Example**
  ```
curl -H 'Authorization: Bearer <token>' -d '{"summoner_id":19887289, "platform":"NA1", "match_id":2165345735}' https://api-staging.vantagesports.com/lolstats/v1/details

{
  "basic": {
    "created": "2016-05-06T18:33:28.275492Z",
    "last_updated": "2016-05-06T18:33:28.275492Z",
    "summonerId": 19887289,
    "platform": "NA1",
    "purchaseState": "UNAVAILABLE",
    "matchCreation": 1461345071374,
    "championId": 48,
    "lane": "TOP",
    "championIndex": 6,
    "mapId": 11,
    "matchDuration": 2038,
    "matchId": 2165345735,
    "matchMode": "CLASSIC",
    "matchType": "MATCHED_GAME",
    "matchVersion": "6.8.140.5213",
    "offMeta": false,
    "opponentChampionId": 68,
    "queueType": "TEAM_BUILDER_DRAFT_RANKED_5x5",
    "role": "SOLO",
    "summonerName": "Imaqtpie",
    "winner": false,
    "tier": "CHALLENGER",
    "division": "I",
    "kills": 6,
    "deaths": 6,
    "assists": 6,
    "teamKills": 22,
    "enemyTeamKills": 29,
    "minionsKilled": 251,
    "csPerMinuteZeroToTen": 6.3,
    "csPerMinuteDiffZeroToTen": -0.2999999999999998,
    "neutralMinionsKilledPerMinute": 0.44160942100098133,
    "neutralMinionsKilledEnemyJunglePerMinute": 0.08832188420019627,
    "goldEarned": 14417,
    "goldZeroToTen": 266.2,
    "goldDiffZeroToTen": 1.3000000000000114,
    "goldTenToTwenty": 424.7,
    "goldDiffTenToTwenty": 50.19999999999999,
    "goldTwentyToThirty": 587.9,
    "goldDiffTwentyToThirty": 317.09999999999997,
    "level6Seconds": 358.061,
    "level6DiffSeconds": 16.61500000000001,
    "wardsPlaced": 16,
    "wardsPlacedPerMinute": 0.4710500490677134,
    "totalDamageDealt": 179848,
    "totalDamageDealtPerMinute": 5294.838076545632,
    "totalDamageDealtToChampions": 21663,
    "totalDamageDealtToChampionsPerMinute": 637.7723258096172,
    "totalDamageTaken": 44514,
    "totalDamageTakenPerMinute": 1310.5201177625122,
    "totalHeal": 17322,
    "totalHealPerMinute": 509.97055937193323
  },
  "advanced": null
}
```
