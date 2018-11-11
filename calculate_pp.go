package main

import (
	"math"

	"github.com/fatih/color"
)

type ppUserMode struct {
	countScores int
	ppTotal     int
}

func opCalculatePP() {
	defer wg.Done()

	const ppQuery = "SELECT scores_relax.userid, pp, scores_relax.play_mode FROM scores_relax INNER JOIN users ON users.id=scores_relax.userid WHERE completed = '3' ORDER BY pp DESC"
	rows, err := db.Query(ppQuery)
	if err != nil {
		queryError(err, ppQuery)
		return
	}

	users := make(map[int]*[4]*ppUserMode)
	var count int

	for rows.Next() {
		if count%1000 == 0 {
			verboseln("> CalculatePP:", count)
		}
		var (
			userid   int
			ppAmt    *float64
			playMode int
		)
		err := rows.Scan(&userid, &ppAmt, &playMode)
		if err != nil {
			queryError(err, ppQuery)
			continue
		}
		if ppAmt == nil {
			continue
		}
		if users[userid] == nil {
			users[userid] = &[4]*ppUserMode{
				new(ppUserMode),
				new(ppUserMode),
				new(ppUserMode),
				new(ppUserMode),
			}
		}
		if users[userid][playMode].countScores > 500 {
			continue
		}
		currentScorePP := round(round(*ppAmt) * math.Pow(0.95, float64(users[userid][playMode].countScores)))
		users[userid][playMode].countScores++
		users[userid][playMode].ppTotal += int(currentScorePP)
		count++
	}
	rows.Close()

	for userid, pps := range users {
		for mode, ppUM := range *pps {
			op("UPDATE users_stats SET pp_"+modeToString(mode)+"_rx = ? WHERE id = ?", ppUM.ppTotal, userid)
		}
	}

	color.Green("> CalculatePP: done!")

	if c.PopulateRedis {
		verboseln("Starting to populate redis")
		go opPopulateRedis()
	}
}

func round(a float64) float64 {
	if a < 0 {
		return math.Ceil(a - 0.5)
	}
	return math.Floor(a + 0.5)
}
