/*
Start local server for development: dev_appserver.py ./
Deploy to GAE: appcfg.py update ./
*/

package banhelper

import (
	"encoding/json"
    "fmt"
    "html/template"
    "io/ioutil"
    "log"
    "net/http"
    "strings"

    "appengine"
    "appengine/urlfetch"
)

var (
	outputTemp = template.Must(template.ParseFiles("ban-output.html"))
	// TODO: fix cache
	// cache map[string]*summonerStats
	mainHTML string
)

// Example HTTP request URL: https://na.api.pvp.net/api/lol/na/v1.4/summoner/by-name/Arkaotics?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c
const (
	key = "99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	summonerId = 30147526
	summonerName = "Arkaotics"

	maxSummoners = 6

	s3 = "SEASON3"
	s4 = "SEASON2014"
	s5 = "SEASON2015"

	rankedSolo = "RANKED_SOLO_5x5"

	getChampion = "https://global.api.pvp.net/api/lol/static-data/na/v1.2/champion/%v?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getSummonerInfo = "https://na.api.pvp.net/api/lol/na/v1.4/summoner/by-name/%v?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getSummonerStats = "https://na.api.pvp.net/api/lol/na/v1.3/stats/by-summoner/%v/ranked?season=%v&api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getLeague = "https://na.api.pvp.net/api/lol/na/v2.5/league/by-summoner/%v/entry?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getMatchHistory = "https://na.api.pvp.net/api/lol/na/v2.2/matchhistory/%v?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getSumMasteries = "https://na.api.pvp.net/api/lol/na/v1.4/summoner/%v/masteries?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
	getSumRunes = "https://na.api.pvp.net/api/lol/na/v1.4/summoner/%v/runes?api_key=99a0db50-1903-4a0d-afa7-bcfc99bc739c"
)

func init() {
    http.HandleFunc("/", root)
    http.HandleFunc("/output", outputHandler)

    // TODO: fix cache
    // cache = make(map[string]*summonerStats)
    if contents, err := ioutil.ReadFile("ban.html"); err != nil {
    	log.Fatalf("Error in loading main page HTML: %v", err)
    } else {
    	mainHTML = string(contents)
    }
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, mainHTML)
}

func outputHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)

	origNames := make([]string, 0)
	names := make([]string, 0)
	
	// Get summoner names from input and create one comma-separated string
	for i := 0; i < maxSummoners; i++ {
		origName := r.FormValue(fmt.Sprintf("summoner%d", i))
		if strings.TrimSpace(origName) == "" { // Whitespace. No summoner to look up.
			origNames = append(origNames, "")
			names = append(names, "")
		} else {
			origNames = append(origNames, origName)
			names = append(names, riotSummonerName(origName))
		}
	}
	allNames := strings.Join(names, ",")

	// Send batch API request to get all summoner IDs
	summIDs := make([]string, 0)

	summInfoURL := fmt.Sprintf(getSummonerInfo, allNames)
	infoResp, err := client.Get(summInfoURL)
	if err != nil {
		log.Fatalf("Error in getting summoner info to retrieve IDs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var si summonerInfo
	if err := json.NewDecoder(infoResp.Body).Decode(&si); err != nil {
		log.Fatalf("Error in decoding summoner info to retrieve IDs: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	for _, n := range names {
		summIDs = append(summIDs, fmt.Sprintf("%d", si[n].ID))
	}
	
	allIDs := strings.Join(summIDs, ",")
	leagueInfoURL := fmt.Sprintf(getLeague, allIDs)
	leagueResp, err := client.Get(leagueInfoURL)
	if err != nil {
		log.Fatalf("Error in getting league info: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	var ld leagueData
	if err := json.NewDecoder(leagueResp.Body).Decode(&ld); err != nil {
		log.Fatalf("Error in decoding league info: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Look up all summoners by their ID and add their summoner stats objects to a map for output page
	asm := make(allStatsMap)
	for j := 0; j < maxSummoners; j++ {
		if origNames[j] == "" { // Whitespace. No summoner to look up.
			continue
		}

		// TODO: fix cache
		// ss := getSummStatsFromCache(names[j])
		// if ss != nil {
		// 	log.Println("Cache entry found")
		// 	asm[j] = ss
		// 	continue
		// }

		// Send API request to get summoner ranked info
		summStatsURL := fmt.Sprintf(getSummonerStats, summIDs[j], s5)
		statsResp, err := client.Get(summStatsURL)
		if err != nil {
			log.Fatalf("Error in getting summoner ranked info for summoner %s: %v", names[j], err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		var summStats summonerStats
		if err := json.NewDecoder(statsResp.Body).Decode(&summStats); err != nil {
		    summStats.Valid = false // Invalid summoner ID (summoner does not exist)
		    summStats.Name = origNames[j]
		    asm[j] = &summStats
		    continue
		} else {
			summStats.Valid = true
		}

		// Add league data to summoner's stats
		var tier string
		var division string
		types := ld[summIDs[j]]
		for _, t := range types {
			if t.Queue == rankedSolo {
				tier = t.Tier
				division = t.Entries[0].Division
			}
		}

		if tier == "" {
			summStats.Rank = "Unranked"
		} else {
			summStats.Rank = tier + " " + division
		}

		// Add additional information to each summoner's champion stats
		champs := summStats.Champions
		for i, champ := range champs {
			stats := champ.Stats
			champURL := fmt.Sprintf(getChampion, champ.ID)
			champResp, err := client.Get(champURL)
			if err != nil {
				log.Fatalf("Error in getting champion information: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			var champ championData
			json.NewDecoder(champResp.Body).Decode(&champ)
			champs[i].Name = champ.Name
			champs[i].FormattedName = formatChampName(champ.Name)
			champs[i].Title = champ.Title

			champs[i].Stats.AverageDeaths = round(float64(stats.TotalDeathsPerSession) / float64(stats.TotalSessionsPlayed), 10)
			champs[i].Stats.AverageKills = round(float64(stats.TotalChampionKills) / float64(stats.TotalSessionsPlayed), 10)
			champs[i].Stats.AverageAssists = round(float64(stats.TotalAssists) / float64(stats.TotalSessionsPlayed), 10)
			champs[i].Stats.WinRate = fmt.Sprintf("%.2f%%", float64(stats.TotalSessionsWon) / float64(stats.TotalSessionsPlayed) * 100)
			champs[i].Stats.KDA = fmt.Sprintf("%.2f", (champs[i].Stats.AverageKills + champs[i].Stats.AverageAssists) / champs[i].Stats.AverageDeaths)
		}
		By(games).Sort(champs)
		if len(summStats.Champions) > 10 {
			summStats.Champions = summStats.Champions[:10]
		}

		summStats.Name = origNames[j]
		asm[j] = &summStats
		// TODO: fix cache
		// putIntoCache(names[j], &summStats)
	}

	if err := outputTemp.Execute(w, asm); err != nil {
		log.Fatalf("Error in executing output page template: %v", err)
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

func getSuggestedBans(asm allStatsMap) []string {
	return []string{"A", "B", "C"}
}

// TODO: fix cache
// func getSummStatsFromCache(name string) *summonerStats {
// 	if ss, ok := cache[name]; ok {
// 		return ss
// 	}
// 	return nil
// }

// func putIntoCache(name string, ss *summonerStats) {
// 	cache[name] = ss
// }