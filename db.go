package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

var db *sql.DB
var db_env_var_name string

const SQL_CREATE_PLUGS = `CREATE TABLE plugs (
id              SERIAL PRIMARY KEY,
s3id            VARCHAR(64) NOT NULL,
owner           VARCHAR(32) NOT NULL,
views           INTEGER NOT NULL,
approved        BOOLEAN NOT NULL
);`

const SQL_CREATE_LOG_TABLE = `CREATE TABLE logs (
time            TIMESTAMP PRIMARY KEY,
severity        INTEGER NOT NULL,
message         TEXT NOT NULL
);`

const SQL_CREATE_PLUG = `INSERT into plugs (s3id, owner, views, approved)
VALUES ($1::text, $2::text, $3::integer, false)`

const SQL_RETRIEVE_APPROVED_PLUGS = `SELECT id, s3id, owner, views FROM plugs WHERE approved=true`

const SQL_RETRIEVE_PLUG_BY_ID = `SELECT s3id, owner, views, approved FROM plugs WHERE id=$1::integer`

const SQL_RETRIEVE_PENDING_PLUGS = `SELECT id, s3id, owner, views, approved FROM plugs WHERE views>=0`

const SQL_SET_PENDING_PLUGS = `UPDATE plugs
SET approved = true
WHERE $1::text LIKE CONCAT('%,',id,',%');`

const SQL_DELETE_PLUG = `DELETE from plugs WHERE id=$1::integer;`

const SQL_INSERT_LOG = `INSERT into logs (time, severity, message)
VALUES ($1, $2::integer, $3::text)`

func reconnectToDB() *sql.DB {
	db_con, err := sql.Open("postgres", os.Getenv(db_env_var_name))
	if err != nil {
		log.Fatal("error connecting to db!")
	}
	return db_con
}

func pingDBAlive() {
	if db.Ping() != nil {
		db = reconnectToDB()
	}
}

func DBInit(env_var_name string) {
	db_env_var_name = env_var_name
	db = reconnectToDB()
	create_table_safe("plugs", SQL_CREATE_PLUGS)
	create_table_safe("logs", SQL_CREATE_LOG_TABLE)
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
		DeletePlug(finalPlug)
		// try again
		return GetPlug()
	}

	return finalPlug
}

func GetPlugById(id int) Plug {
	rows, err := db.Query(SQL_RETRIEVE_PLUG_BY_ID, id)

	if err != nil {
		log.Fatal(err)
	}

	var obj Plug
	obj.ID = id
	for rows.Next() {
		err = rows.Scan(&obj.S3ID, &obj.Owner, &obj.ViewsRemaining, &obj.Approved)

		if err != nil {
			log.Error(err)
		}

		// Return after first result
		return obj
	}

	log.Fatal("We should not be able to reach this point!")
	return obj
}

func DeletePlug(plug Plug) {
	_, err := db.Exec(SQL_DELETE_PLUG, plug.ID)
	if err != nil {
		log.Error(err)
	}
	S3DelFile(plug)

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

func AddLog(severity int, message string) {
	_, err := db.Exec(
		SQL_INSERT_LOG,
		time.Now(),
		severity,
		message)

	if err != nil {
		log.Error(err)
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
