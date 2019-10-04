# lolusers

[![wercker status](https://app.wercker.com/status/ec89073a4937eff48d6e6635bdbb6134/m "wercker status")](https://app.wercker.com/project/bykey/ec89073a4937eff48d6e6635bdbb6134)

This API deals with vantagesports.gg customers, managing things like point balances, summoner associations, etc.

Method    | Endpoint                           | Description                         | Data Format
|---------|------------------------------------|-------------------------------------|---------------------------------------------
| POST    | /lolusers/v1/ListRiotSummoner      | retrieve a Summoner object via the Riot API   | { riot_summoner: { id: summonerId or name: summonerName, region: region }}                                                                 
| POST    | /lolusers/v1/Create                | create an LolUser                   | {lol_user: {user_id: userId, summoner_id: summonerId, region: region }}
| POST    | /lolusers/v1/Update                | update an LolUser                   | {lol_user: {user_id: userId, summoner_id: summonerId, region: region }}
| POST    | /lolusers/v1/List                  | retrieve a list of LolUsers, can send userId, summonerId, or region in request.  Only admins can send an empty request to retrieve all LolUsers | {user_id: userId, summoner_id: summonerId, region: region }
| POST    | /lolusers/v1/ListChampionsByRegion | get all Champions from the Riot API given a region | { id: region }
| POST    | /lolusers/v1/AdjustVantagePoints   | adjust the vantage point balance of an loluser.  Only admins are authorized | { user_id: userId, amount: vantagePoints }
| POST    | /lolusers/v1/PurchaseMatch         | purchase a match so advanced stats will be analyzed | { match_id: matchId, summoner_id: summonerId, platform: platform, user_id: userId }

