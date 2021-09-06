package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type DBConnection struct {
	app    *PlugApplication
	con    *sql.DB
	db_uri string
}

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

func (c *DBConnection) Init(app *PlugApplication, db_uri string) {
	c.app = app
	c.db_uri = db_uri
	c.reconnectToDB()
	c.create_table_safe("plugs", SQL_CREATE_PLUGS)
	c.create_table_safe("logs", SQL_CREATE_LOG_TABLE)
}

func (c *DBConnection) reconnectToDB() {
	db_con, err := sql.Open("postgres", c.db_uri)
	if err != nil {
		log.Fatal("error connecting to db!")
	}
	c.con = db_con
}

func (c DBConnection) pingDBAlive() {
	err := c.con.Ping()
	if err != nil {
		log.Info("failed to ping db!")
		log.Info(err)
		c.reconnectToDB()
	}
}

func (c DBConnection) create_table_safe(name, sql string) {
	c.pingDBAlive()
	rows, err := c.con.Query("SELECT 1::integer FROM pg_tables WHERE schemaname = 'public' AND tablename = $1::text;",
		name)
	if err != nil {
		log.Error(err)
	}
	if !rows.Next() {
		_, err = c.con.Exec(sql)
		if err != nil {
			log.Fatal(err)
		}
	} else {
	}
}

func (c DBConnection) GetPlug() Plug {
	rows, err := c.con.Query(SQL_RETRIEVE_APPROVED_PLUGS)

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
		_, err = c.con.Exec("UPDATE plugs SET views=$2::integer WHERE id=$1::integer;",
			finalPlug.ID, finalPlug.ViewsRemaining)
		if err != nil {
			log.Error(err)
		}
	}
	if finalPlug.ViewsRemaining == 0 {
		c.DeletePlug(finalPlug)
		// try again
		return c.GetPlug()
	}

	return finalPlug
}

func (c DBConnection) GetPlugById(id int) Plug {
	rows, err := c.con.Query(SQL_RETRIEVE_PLUG_BY_ID, id)

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

func (c DBConnection) DeletePlug(plug Plug) {
	_, err := c.con.Exec(SQL_DELETE_PLUG, plug.ID)
	if err != nil {
		log.Error(err)
	}
	c.app.s3.DelFile(plug)

}

func (c DBConnection) GetPendingPlugs() []Plug {
	rows, err := c.con.Query(SQL_RETRIEVE_PENDING_PLUGS)

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

func (c DBConnection) GetUserPlugs(user string) []Plug {
	rows, err := c.con.Query(SQL_RETRIEVE_PENDING_PLUGS)

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
		if obj.Owner == user {
			plugs = append(plugs, obj)
		}
	}

	return plugs
}

func (c DBConnection) SetPendingPlugs(approvedList []string) {
	_, err := c.con.Exec("UPDATE plugs SET approved = false;")
	if err != nil {
		log.Fatal(err)
	}

	stringList := "," + strings.Join(approvedList, ",") + ","
	_, err = c.con.Exec(SQL_SET_PENDING_PLUGS, stringList)

	if err != nil {
		log.Fatal(err)
	}
}

func (c DBConnection) AddLog(severity int, message string) {
	_, err := c.con.Exec(
		SQL_INSERT_LOG,
		time.Now(),
		severity,
		message)

	if err != nil {
		log.Error(err)
	}
}

func (c DBConnection) MakePlug(plug Plug) {
	_, err := c.con.Exec(
		SQL_CREATE_PLUG,
		plug.S3ID,
		plug.Owner,
		plug.ViewsRemaining,
	)
	if err != nil {
		log.Error(err)
	}
}
