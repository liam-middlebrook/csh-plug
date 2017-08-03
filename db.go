package main

import (
    "database/sql"
    _ "github.com/lib/pq"
    log "github.com/sirupsen/logrus"
)

var db *sql.DB

const SQL_CREATE_PLUGS = `CREATE TABLE plugs (
id              SERIAL PRIMARY KEY,
s3id            VARCHAR(64) NOT NULL,
owner           VARCHAR(32) NOT NULL,
views           INTEGER NOT NULL
);`

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
        _, err = db.Exec(sql);
        if err != nil {
            log.Fatal(err)
        }
    } else {
    }
}

func GetPlug() Plug {
    rows, err := db.Query("SELECT id, s3id, owner, views FROM plugs")

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
        _, err = db.Exec("DELETE from plugs WHERE id=$1::integer;", finalPlug.ID)
        if err != nil {
            log.Error(err)
        }
    }

    return finalPlug
}
