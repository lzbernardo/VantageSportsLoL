package scrape

import (
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	BaseURL string = "http://lol.esportspedia.com"
)

type Pick struct {
	Champion string `json:"champ"`
	Position string `json:"pos"`
}

type PickBan struct {
	Id       int    `json:"i"`
	Season   string `json:"season"`
	Round    string `json:"round"`
	Team1    string `json:"team1"`
	Team2    string `json:"team2"`
	Team1Won bool   `json:"team1won"`

	Team1Bans []string `json:"team1bans"`
	Team2Bans []string `json:"team2bans"`

	Team1Picks []Pick `json:"team1picks"`
	Team2Picks []Pick `json:"team2picks"`
}

func scrapePickRow(pickRow *goquery.Selection) (Pick, Pick) {
	pick1Cells := pickRow.Children()
	team1Pick := strings.TrimSpace(pick1Cells.First().Text())
	team2Pick := strings.TrimSpace(pick1Cells.Last().Text())

	team1Pos, found := pick1Cells.Eq(1).Find("span").Attr("title")
	if !found {
		log.Println("title not found in pick row")
	}
	team2Pos, found := pick1Cells.Eq(2).Find("span").Attr("title")
	if !found {
		log.Println("title not found in pick row")
	}

	pick1 := Pick{
		Champion: team1Pick,
		Position: team1Pos,
	}
	pick2 := Pick{
		Champion: team2Pick,
		Position: team2Pos,
	}
	return pick1, pick2
}

func ScrapePicksAndBans() ([]PickBan, error) {
	urls, err := GetSeasonsURLs(BaseURL)
	if err != nil {
		return nil, err
	}

	pbDataset := []PickBan{}

	for _, url := range urls {
		if url == "" {
			log.Println("empty url")
			continue
		}
		pickAndBansURL, err := GetPicksAndBansURL(BaseURL, url)
		if err != nil {
			return nil, err
		}
		if pickAndBansURL == "" {
			log.Println("empty picks and bans url")
			continue
		}

		pbs := ScrapeLCS(pickAndBansURL)
		pbDataset = append(pbDataset, pbs...)
	}
	return pbDataset, nil
}

func ScrapeLCS(url string) []PickBan {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatal(err)
	}

	pbs := []PickBan{}

	season := doc.Find("h1.firstHeading span").Text()

	doc.Find("h2").Each(func(i int, s *goquery.Selection) {
		week := strings.TrimSpace(s.Find("span.mw-headline").Text())
		if week == "" {
			return
		}

		s.Next().Next().Find("table.prettytable").Each(func(i int, sTable *goquery.Selection) {
			team1Bans := []string{}
			team2Bans := []string{}

			team1 := sTable.Find("b").First()
			team1Won := strings.Contains(team1.Parent().Next().AttrOr("style", ""), "background-color")
			team2 := sTable.Find("b").Last()

			if team1.Text() == "" || team2.Text() == "" {
				return
			}

			// ban rows
			// Team 1 Ban | "Bans" | Team 2 Ban
			sTable.Find("tr.allPnBs i").Each(func(i int, sRow *goquery.Selection) {
				// left of Ban
				team1Ban := strings.TrimSpace(sRow.Parent().Prev().Text())
				if team1Ban != "Loss of Ban" {
					team1Bans = append(team1Bans, team1Ban)
				}

				// right of Ban
				team2Ban := strings.TrimSpace(sRow.Parent().Next().Text())
				if team2Ban != "Loss of Ban" {
					team2Bans = append(team2Bans, team2Ban)
				}

				// third and final ban
				if i == 2 {
					team1Picks := []Pick{}
					team2Picks := []Pick{}

					pick1Row := sRow.Parent().Parent().Next()
					team1Pick, team2Pick := scrapePickRow(pick1Row)
					team1Picks = append(team1Picks, team1Pick)
					team2Picks = append(team2Picks, team2Pick)

					if team1Pick.Champion == "" || team2Pick.Champion == "" {
						return
					}

					pick2Row := pick1Row.Next()
					team1Pick, team2Pick = scrapePickRow(pick2Row)
					team1Picks = append(team1Picks, team1Pick)
					team2Picks = append(team2Picks, team2Pick)

					pick3Row := pick2Row.Next()
					team1Pick, team2Pick = scrapePickRow(pick3Row)
					team1Picks = append(team1Picks, team1Pick)
					team2Picks = append(team2Picks, team2Pick)

					pick4Row := pick3Row.Next()
					team1Pick, team2Pick = scrapePickRow(pick4Row)
					team1Picks = append(team1Picks, team1Pick)
					team2Picks = append(team2Picks, team2Pick)

					pick5Row := pick4Row.Next()
					team1Pick, team2Pick = scrapePickRow(pick5Row)
					team1Picks = append(team1Picks, team1Pick)
					team2Picks = append(team2Picks, team2Pick)

					pickBan := PickBan{
						Season:     season,
						Round:      week,
						Team1:      team1.Text(),
						Team2:      team2.Text(),
						Team1Won:   team1Won,
						Team1Bans:  team1Bans,
						Team2Bans:  team2Bans,
						Team1Picks: team1Picks,
						Team2Picks: team2Picks,
					}

					pbs = append(pbs, pickBan)
				}
			})
		})
	})
	return pbs
}

func GetSeasonsURLs(baseURL string) ([]string, error) {
	urls := []string{}
	doc, err := goquery.NewDocument(baseURL)
	if err != nil {
		return nil, err
	}

	doc.Find("div.tab-Tourney span a").Each(func(i int, s *goquery.Selection) {
		tourney, found := s.Attr("href")
		if !found {
			log.Println("href not found")
			return
		}
		if tourney == "" {
			return
		}

		urls = append(urls, baseURL+tourney)
	})
	return urls, nil
}

func GetPicksAndBansURL(baseURL, tournamentURL string) (string, error) {
	doc, err := goquery.NewDocument(tournamentURL)
	if err != nil {
		return "", err
	}

	url := ""
	doc.Find("div#bodyContent td a").Each(func(i int, s *goquery.Selection) {
		picksAndBans, found := s.Attr("href")
		if !found {
			log.Println("href not found")
			return
		}
		if strings.Contains(picksAndBans, "Picks_and_Bans") {
			log.Printf("tournament to scrape: %s\n", picksAndBans)
			url = baseURL + picksAndBans
		}
	})
	return url, nil
}
