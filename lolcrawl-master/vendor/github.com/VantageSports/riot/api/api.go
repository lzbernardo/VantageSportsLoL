package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/VantageSports/riot"
)

// Set to true to see every request url requested by the API
var Verbose = false

type Api struct {
	key         string
	region      riot.Region
	rateLimiter RateLimiter
}

func NewAPI(key string, region riot.Region, rate CallRate) *Api {
	return &Api{
		key:         key,
		region:      region,
		rateLimiter: NewDurationLimiter(rate.CallsPer, rate.Dur),
	}
}

type APIs map[riot.Region]*Api

func (a APIs) Region(r string) (*Api, error) {
	region := riot.RegionFromString(r)
	if region == "" {
		return nil, fmt.Errorf("unknown region: %s", r)
	}
	res := a[region]
	if res == nil {
		return nil, fmt.Errorf("no api registered for region: %s", region)
	}
	return res, nil
}

func (a APIs) Platform(p string) (*Api, error) {
	platform := riot.PlatformFromString(p)
	if platform == "" {
		return nil, fmt.Errorf("unknown platform: %s", p)
	}
	region := riot.RegionFromPlatform(platform)
	return a.Region(string(region))
}

func NewAPIs(key string, rate CallRate) APIs {
	res := APIs{}
	for _, region := range riot.AllRegions() {
		res[region] = NewAPI(key, region, rate)
	}
	return res
}

func (a *Api) getJSON(url string, v interface{}) error {
	a.rateLimiter.Wait()
	err := getJSON(url, v)
	a.rateLimiter.Complete()

	return err
}

func (a *Api) Region() riot.Region {
	return a.region
}

func getJSON(url string, v interface{}) error {
	if Verbose {
		fmt.Println(url)
	}
	resp, err := http.Get(url)
	if err != nil {
		if Verbose {
			fmt.Println(err)
		}
		return err
	}

	// Read the response body and close right away so connection can be reused.
	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// Some servers (cough, replay.gg, cough) return bad json strings
	endIndex := len(data)
	for endIndex > 0 && data[endIndex-1] == 0 {
		endIndex--
	}
	data = data[0:endIndex]

	if resp.StatusCode != 200 {
		rateLimitTypeStr := resp.Header.Get("X-Rate-Limit-Type")
		retryAfterStr := resp.Header.Get("Retry-After")
		return NewAPIError(resp.StatusCode, rateLimitTypeStr, retryAfterStr)
	}

	if err = json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}

func (a *Api) baseURL() string {
	return fmt.Sprintf("https://%v.api.pvp.net", a.region)
}

func (a *Api) globalURL() string {
	return "https://global.api.pvp.net"
}

func (a *Api) addParams(urlStr string, params ...string) (string, error) {
	if len(params)%2 != 0 {
		return urlStr, fmt.Errorf("len(params) must be even")
	}

	u, err := url.Parse(urlStr)
	q := u.Query()
	for i := 0; i < len(params); i += 2 {
		q.Set(params[i], params[i+1])
	}
	if a.key != "" {
		q.Set("api_key", a.key)
	}

	u.RawQuery = q.Encode()
	return u.String(), err
}

func spectatorBaseURL(platformID string) string {
	platformID = strings.ToLower(platformID)
	switch platformID {
	case "na1":
		return "http://spectator.na.lol.riotgames.com"
	case "euw1":
		return "http://spectator.euw1.lol.riotgames.com"
	case "eun1":
		return "http://spectator.eu.lol.riotgames.com:8088"
	case "jp":
		return "http://spectator.jp1.lol.riotgames.com"
	case "kr":
		return "http://spectator.kr.lol.riotgames.com"
	case "oc1":
		return "http://spectator.oc1.lol.riotgames.com"
	case "br1":
		return "http://spectator.br.lol.riotgames.com"
	case "la1":
		return "http://spectator.la1.lol.riotgames.com"
	case "la2":
		return "http://spectator.la2.lol.riotgames.com"
	case "ru":
		return "http://spectator.ru.lol.riotgames.com"
	case "tr1":
		return "http://spectator.tr.lol.riotgames.com"
	case "pbe1":
		return "http://spectator.pbe1.lol.riotgames.com:8088"
	default:
		return ""
	}
}
