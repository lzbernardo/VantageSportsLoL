package main

import (
	"fmt"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/api/sheets/v4"

	"github.com/VantageSports/common/credentials/google"
)

func main() {
	ctx := context.Background()

	creds := google.MustEnvCreds("vs-main", sheets.SpreadsheetsReadonlyScope)

	srv, err := sheets.New(creds.Conf.Client(ctx))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	spreadsheetId := "1ifsGvHXf4sSoUhxaPONwF2PwLYGN_MBifyY5Q9SM62w"
	sheetResp, err := srv.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		log.Fatalf("Unable to spreadsheet from sheet. %v", err)
	}
	for _, s := range sheetResp.Sheets {
		fmt.Printf("Id: %v, Title: %v\n", s.Properties.SheetId, s.Properties.Title)

		readRange := fmt.Sprintf("%v!A2:A", s.Properties.Title)
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet. %v", err)
		}

		if len(resp.Values) > 0 {
			for _, row := range resp.Values {
				if len(row) > 0 {
					fmt.Printf("%s\n", row[0])
				}
			}
		} else {
			fmt.Print("No data found.")
		}
	}
}
