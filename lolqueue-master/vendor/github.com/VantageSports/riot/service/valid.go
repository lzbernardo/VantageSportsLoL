package service

import "fmt"

func (sr *SummonerIDRequest) Valid(max int) error {
	if len(sr.Ids) > max {
		return fmt.Errorf("cannot request more than %d ids per request", max)
	}
	return nil
}

func (sr *SummonerNameRequest) Valid(max int) error {
	if len(sr.Names) > max {
		return fmt.Errorf("cannot request more than %d names per request", max)
	}
	return nil
}
