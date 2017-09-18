package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"strings"
)

var db *sql.DB

const SQL_CREATE_PLUGS = `CREATE TABLE plugs (
id              SERIAL PRIMARY KEY,
s3id            VARCHAR(64) NOT NULL,
owner           VARCHAR(32) NOT NULL,
views           INTEGER NOT NULL,
approved        BOOLEAN NOT NULL
);`

const SQL_CREATE_PLUG = `INSERT into plugs (s3id, owner, views, approved)
VALUES ($1::text, $2::text, $3::integer, false)`

const SQL_RETRIEVE_APPROVED_PLUGS = `SELECT id, s3id, owner, views FROM plugs WHERE approved=true`

const SQL_RETRIEVE_PENDING_PLUGS = `SELECT id, s3id, owner, views, approved FROM plugs WHERE views>=0`

const SQL_SET_PENDING_PLUGS = `UPDATE plugs
SET approved = true
WHERE $1::text LIKE CONCAT('%,',id,',%');`

const SQL_DELETE_PLUG = `DELETE from plugs WHERE id=$1::integer;`

func DBInit(db_uri string) {
	var err error
	db, err = sql.Open("postgres", db_uri)
	if err != nil {
		log.Fatal("error connecting to db!")
	}

	create_table_safe("plugs", SQL_CREATE_PLUGS)
}

func create_table_safe(name, sql string) {
	rows, err := db.Query("SELECT 1::integer FROM pg_tables WHERE schemaname = 'public' AND tablename = $1::text;",
		name)
	if err != nil {
		log.Error(err)
	}
	if !rows.Next() {
		_, err = db.Exec(sql)
		if err != nil {
			log.Fatal(err)
		}
	} else {
	}
}

func GetPlug() Plug {
	rows, err := db.Query(SQL_RETRIEVE_APPROVED_PLUGS)

	if err != nil {
		log.Fatal(err)
	}

	var plugs []Plug
	for rows.Next() {
		var obj Plug
		err = rows.Scan(&obj.ID, &obj.S3ID, &obj.Owner, &obj.ViewsRemaining)

		if err != nil {
			log.Error(err)
		}

		plugs = append(plugs, obj)
	}
	finalPlug := ChoosePlug(plugs)

	if finalPlug.ViewsRemaining > 0 {
		finalPlug.ViewsRemaining -= 1
		_, err = db.Exec("UPDATE plugs SET views=$2::integer WHERE id=$1::integer;",
			finalPlug.ID, finalPlug.ViewsRemaining)
		if err != nil {
			log.Error(err)
		}
	}
	if finalPlug.ViewsRemaining == 0 {
		_, err = db.Exec(SQL_DELETE_PLUG, finalPlug.ID)
		if err != nil {
			log.Error(err)
		}
		S3DelFile(finalPlug)

		// try again
		return GetPlug()
	}

	return finalPlug
}

func GetPendingPlugs() []Plug {
	rows, err := db.Query(SQL_RETRIEVE_PENDING_PLUGS)

	if err != nil {
		log.Fatal(err)
	}

	var plugs []Plug
	for rows.Next() {
		var obj Plug
		err = rows.Scan(&obj.ID, &obj.S3ID, &obj.Owner, &obj.ViewsRemaining, &obj.Approved)

		if err != nil {
			log.Error(err)
		}
		plugs = append(plugs, obj)
	}

	return plugs
}

func SetPendingPlugs(approvedList []string) {
	_, err := db.Exec("UPDATE plugs SET approved = false;")
	if err != nil {
		log.Fatal(err)
	}

	stringList := "," + strings.Join(approvedList, ",") + ","
	_, err = db.Exec(SQL_SET_PENDING_PLUGS, stringList)

	if err != nil {
		log.Fatal(err)
	}
}

func MakePlug(plug Plug) {
	_, err := db.Exec(
		SQL_CREATE_PLUG,
		plug.S3ID,
		plug.Owner,
		plug.ViewsRemaining,
	)
	if err != nil {
		log.Error(err)
	}
}
