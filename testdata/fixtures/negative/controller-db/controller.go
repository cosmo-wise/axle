package controller

import "database/sql"

func Handle(db *sql.DB) error { return db.Ping() }
