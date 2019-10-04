package main

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/net/context"
	"google.golang.org/api/sheets/v4"
)

func getMatchIdsFromSpreadsheet(ctx context.Context, srv *sheets.Service, spreadsheetId string) (map[string][]int64, error) {
	results := map[string][]int64{}

	sheetResp, err := srv.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		log.Fatalf("Unable to spreadsheet from sheet. %v", err)
	}
	for _, s := range sheetResp.Sheets {
		readRange := fmt.Sprintf("%v!A2:A", s.Properties.Title)
		resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet. %v", err)
		}

		if len(resp.Values) > 0 {
			for _, row := range resp.Values {
				if len(row) > 0 {
					strVal, ok := row[0].(string)
					if !ok {
						log.Fatalf("cannot convert %v to string", row[0])
					}

					intVal, err := strconv.ParseInt(strVal, 10, 64)
					if err != nil {
						log.Fatal(err)
					}
					results[s.Properties.Title] = append(results[s.Properties.Title], intVal)
				}
			}
		} else {
			fmt.Println("No data found.")
		}
	}
	return results, nil
}
