package service

import (
	. "github.com/martinlebeda/taskmaster/model"
	"github.com/martinlebeda/taskmaster/termout"
	"strconv"
	"strings"
	"time"
)

func TmrSet(replaceTag bool, tag, dateOpt, timeArg, title string) {
	goal, err := time.Parse("2006-01-02 15:04", dateOpt+" "+timeArg)
	CheckErr(err)

	insertNewTimer(replaceTag, tag, title, goal)
}

func TmrAdd(replaceTag bool, tag, duration, title string) {
	parseDuration, err := time.ParseDuration(duration)
	CheckErr(err)

	goal := time.Now().Add(parseDuration)

	insertNewTimer(replaceTag, tag, title, goal)
}

func insertNewTimer(replaceTag bool, tag, title string, goal time.Time) {
	termout.Verbose("New goal for ", tag, " - ", title, " set to ", goal.String())
	db := OpenDB()

	if tag != "" && replaceTag {
		stmt, err := db.Prepare("delete from timer where tag = ?")
		CheckErr(err)
		_, err = stmt.Exec(tag)
		CheckErr(err)
	}

	stmt, err := db.Prepare("INSERT INTO timer(note, goal, tag) values(?,?,?)")
	CheckErr(err)
	_, err = stmt.Exec(title, goal, tag)
	CheckErr(err)
	termout.Verbose("New timer inserted")
}

func TmrDel(tmDeleteByName, tmDeleteByTag bool, args []string) {
	// sql by field
	sql := ""
	if tmDeleteByTag {
		sql = "delete from timer where tag in ('" + strings.Join(args, "','") + "')"
	} else if tmDeleteByName {
		sql = "delete from timer where note in ('" + strings.Join(args, "','") + "')"
	} else {
		sql = "delete from timer where rowid in (" + strings.Join(args, ",") + ")"
	}

	// execute delete
	db := OpenDB()
	stmt, err := db.Prepare(sql)
	CheckErr(err)
	_, err = stmt.Exec()
	CheckErr(err)
	termout.Verbose("Timer deleted: ", strings.Join(args, ","))
}

func TmrUpdate(note string, goal time.Time, ids []string) {
	sql := "update timer set"

	// add parameters
	var setSql []string
	var argSql []interface{}
	if note != "" {
		setSql = append(setSql, "note = ?")
		argSql = append(argSql, note)
	}
	if !goal.IsZero() {
		setSql = append(setSql, "goal = ?")
		argSql = append(argSql, goal)
	}

	sql += "set " + strings.Join(setSql, ", ")
	sql += " where rowid in (" + strings.Join(ids, ",") + ")"

	// execute update
	db := OpenDB()
	stmt, err := db.Prepare(sql)
	CheckErr(err)
	_, err = stmt.Exec(argSql...)
	CheckErr(err)
	termout.Verbose("Timer updated: ", strings.Join(ids, ","))
}

func TmrGetDistance(pastOpt, nextOpt bool, tag string) []TimerDistance {
	db := OpenDB()
	sql := "select rowid, distance, goal, CASE WHEN tag IS NULL THEN '' ELSE tag END, note from timer_distance where goal is not null "

	if pastOpt {
		sql += " and distance < 0 "
	}
	if nextOpt {
		sql += " and distance > 0 "
	}

	if tag != "" {
		sql += " and tag '" + tag + "' "
	}

	sql += " order by distance "

	if nextOpt {
		sql += " limit 1 "
	}

	rows, err := db.Query(sql)
	CheckErr(err)

	var result []TimerDistance
	for rows.Next() {
		var timerDistance TimerDistance
		rows.Scan(&timerDistance.Rowid, &timerDistance.Distance, &timerDistance.Goal, &timerDistance.Tag, &timerDistance.Note)
		result = append(result, timerDistance)
	}

	return result
}

func TmrClean(deleteAll bool) {
	db := OpenDB()

	sql := "delete from timer"
	if !deleteAll {
		sql += " where rowid in (select rowid from timer_distance where distance < 0)"
	}

	result, err := db.Exec(sql)
	CheckErr(err)
	count, err := result.RowsAffected()
	CheckErr(err)
	termout.Verbose("Count of deleted timers: ", strconv.FormatInt(count, 10))
}

func TmrListAfterChange() {
	termout.EmptyLineOut()
	timerDistances := TmrGetDistance(false, false, "")
	termout.TmrListDistance(timerDistances, false)
}
