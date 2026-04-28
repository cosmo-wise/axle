package main

import (
	"context"
	"log"
	"net/http"
	"os"

	appcatalog "github.com/Fel1xKan/axle/testdata/examples/generated-backend/catalog"
	"github.com/Fel1xKan/axle/testdata/examples/generated-backend/internal/app"
	axlesqlite "github.com/Fel1xKan/axle/pkg/axle/sqlite"
)

func main() {
	ctx := context.Background()
	dsn := "file:axle.db"
	if len(os.Args) > 1 {
		dsn = os.Args[1]
	}
	db, err := axlesqlite.Open(ctx, dsn, appcatalog.Catalog)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(":8080", app.New(db)))
}
