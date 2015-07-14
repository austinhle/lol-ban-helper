package banhelper

import (
    "sort"
    "strings"
)

var (
	games = func(c1, c2 *champion) bool {
		return c1.Stats.TotalSessionsPlayed > c2.Stats.TotalSessionsPlayed
	}
)

type By func(c1, c2 *champion) bool

func (by By) Sort(champs []champion) {
	ss := &summonerStats{
		Champions: champs,
		by:        by,
	}
	sort.Sort(ss)
}

type championData struct {
	Name string
	Title string
}

type summonerInfo map[string]struct {
	ID int
	Name string
}

// Implements the sort.Interface
type summonerStats struct {
	Name string
	Rank string
	Champions []champion
	Valid bool
	by func(c1, c2 *champion) bool
}

type allStatsMap map[int]*summonerStats

func (s *summonerStats) Len() int {
	return len(s.Champions)
}

func (s *summonerStats) Swap(i, j int) {
	s.Champions[i], s.Champions[j] = s.Champions[j], s.Champions[i]
}

func (s *summonerStats) Less(i, j int) bool {
	return s.by(&s.Champions[i], &s.Champions[j])
}

type champion struct {
	ID int
	Name string
	FormattedName string
	Title string
	Stats struct {
		TotalChampionKills int
		TotalAssists int
		TotalDeathsPerSession int
		RankedSoloGamesPlayed int
		RankedPremadeGamesPlayed int
		NormalGamesPlayed int
		TotalSessionsPlayed int
		TotalSessionsWon int
		AverageDeaths float64
		AverageKills float64
		AverageAssists float64
		WinRate string
		KDA string
	}
}

type leagueData map[string][]leagueType

type leagueType struct {
	Queue string
	Entries []struct {
		Division string
		Wins int
	}
	Tier string
}

type matchHistoryData struct {
	
}

type masteriesData struct {

}

type runesData struct {

}

func round(n float64, p float64) float64 {
	return float64(int(n * p)) / p
}

func riotSummonerName(name string) string {
	return strings.Replace(strings.ToLower(name), " ", "", -1)
}

func formatChampName(name string) string {
	if name == "Wukong" {
		return "MonkeyKing"
	} else if strings.Contains(name, " ") {
		return strings.Replace(name, " ", "", -1)
	} else if strings.Contains(name, "'") {
		if name == "Cho'Gath" {
			return "Chogath"
		} else if name == "Vel'Koz" {
			return "Velkoz"
		} else if name == "Kha'Zix" {
			return "Khazix"
		} else {
			return strings.Replace(name, "'", "", -1)
		}
	} else if strings.Contains(name, ". ") {
		return strings.Replace(name, ". ", "", -1)
	} else {
		return strings.Title(strings.ToLower(name))
	}
}