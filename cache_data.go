package main

import (
	"strconv"

	"github.com/fatih/color"
	"zxq.co/ripple/ocl"
)

type s struct {
	rankedScore int64
	totalHits   int64
	level       int
}

func opCacheData() {
	defer wg.Done()
	// get data
	const fetchQuery = `
	SELECT
		users.id as user_id, users.username, scores_relax.play_mode,
		scores_relax.score, scores_relax.completed, scores_relax.300_count,
		scores_relax.100_count, scores_relax.50_count
	FROM scores_relax
	INNER JOIN users ON users.id=scores_relax.userid`
	rows, err := db.Query(fetchQuery)
	if err != nil {
		queryError(err, fetchQuery)
		return
	}

	// set up end map where all the data is
	data := make(map[int]*[4]*s)

	count := 0

	// analyse every result row of fetchQuery
	for rows.Next() {
		if count%1000 == 0 {
			verboseln("> CacheData:", count)
		}
		var (
			uid       int
			username  string
			playMode  int
			score     int64
			completed int
			count300  int
			count100  int
			count50   int
		)
		err := rows.Scan(&uid, &username, &playMode, &score, &completed, &count300, &count100, &count50)
		if err != nil {
			queryError(err, fetchQuery)
			continue
		}
		// silently ignore invalid modes
		if playMode > 3 || playMode < 0 {
			continue
		}
		// create key in map if not already existing
		if _, ex := data[uid]; !ex {
			data[uid] = &[4]*s{}
			for i := 0; i < 4; i++ {
				data[uid][i] = &s{}
			}
		}
		// if the score counts as completed and top score, add it to the ranked score sum
		if c.CacheRankedScore && completed == 3 {
			data[uid][playMode].rankedScore += score
		}
		// add to the number of totalhits count of {300,100,50} hits
		if c.CacheTotalHits {
			data[uid][playMode].totalHits += int64(count300) + int64(count100) + int64(count50)
		}
		count++
	}
	rows.Close()

	if c.CacheLevel {
		const totalScoreQuery = "SELECT id, total_score_std_rx, total_score_taiko_rx, total_score_ctb_rx, total_score_mania FROM users_stats"
		rows, err := db.Query(totalScoreQuery)
		if err != nil {
			queryError(err, totalScoreQuery)
			return
		}
		count = 0
		for rows.Next() {
			if count%100 == 0 {
				verboseln("> CacheLevel:", count)
			}
			var (
				id    int
				std   int64
				taiko int64
				ctb   int64
				mania int64
			)
			err := rows.Scan(&id, &std, &taiko, &ctb, &mania)
			if err != nil {
				queryError(err, totalScoreQuery)
				continue
			}
			if _, ex := data[id]; !ex {
				data[id] = &[4]*s{}
				for i := 0; i < 4; i++ {
					data[id][i] = &s{}
				}
			}
			data[id][0].level = ocl.GetLevel(std)
			data[id][1].level = ocl.GetLevel(taiko)
			data[id][2].level = ocl.GetLevel(ctb)
			data[id][3].level = ocl.GetLevel(mania)
			count++
		}
		rows.Close()
	}
	for k, v := range data {
		if v == nil {
			continue
		}
		for modeInt, modeData := range v {
			if modeData == nil {
				continue
			}
			var setQ string
			var params []interface{}
			if c.CacheRankedScore {
				setQ += "ranked_score_" + modeToString(modeInt) + " = ?"
				params = append(params, (*modeData).rankedScore)
			}
			if c.CacheTotalHits {
				if setQ != "" {
					setQ += ", "
				}
				setQ += "total_hits_" + modeToString(modeInt) + " = ?"
				params = append(params, (*modeData).totalHits)
			}
			if c.CacheLevel {
				if setQ != "" {
					setQ += ", "
				}
				setQ += "level_" + modeToString(modeInt) + "_rx" + " = ?"
				params = append(params, (*modeData).level)
			}
			if setQ != "" {
				params = append(params, k)
				op("UPDATE users_stats SET "+setQ+" WHERE id = ?", params...)
			}
		}
	}
	color.Green("> CacheData: done!")
}

var modes = [...]string{
	"std",
	"taiko",
	"ctb",
	"mania",
}

func modeToString(modeID int) string {
	if modeID < len(modes) {
		return modes[modeID]
	}
	return strconv.Itoa(modeID)
}
