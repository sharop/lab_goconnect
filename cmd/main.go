package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/cockroachdb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sharop/lab_goconnect/internal/server"

	"log"
	"time"
	"xorm.io/xorm"
)

/*var settings = cockroachdb.ConnectionURL{
	Database: `example`,
	Host: `localhost`,
	User: `root`,
}*/

var Engine *xorm.Engine

type User struct {
	Id int64
	Name string
	Salt string
	Age int
	Email string `xorm:"varchar(254)"`
	Passwd string `xorm:"varchar(200)"`
	Created time.Time `xorm:"created"`
	Updated time.Time `xorm:"updated"`
}

var (
	Db *sql.DB
	ctx context.Context
)
func InitStruct(name string) {
	// 1. Create connection
	// 2. Create Wallet Info
	// 3. Create DBCollection
	// 4. Create Folder structure

	var connStringdodo  = "postgres://root:@localhost:26257/example?sslmode=disable"
	engine, err := xorm.NewEngine("postgres", connStringdodo)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	if err:= engine.Ping(); err != nil{
		log.Fatal(err)
	}
	defer engine.Close()
	_,err = engine.Exec("CREATE DATABASE IF NOT EXISTS "+name)
	if err != nil {
		panic(err)
	}
	engine.Close()
	connStringdodo  = fmt.Sprintf( `postgres://root:@localhost:26257/%s?sslmode=disable`,name)
	engine, err = xorm.NewEngine("postgres", connStringdodo)
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}
	if err:= engine.Ping(); err != nil{
		log.Fatal(err)
	}
	//defer engine.Close()
	Db = engine.DB().DB

}


func main()  {

	srv:= server.NewHTTPServer(":9010")
	log.Fatal(srv.ListenAndServe())

	/*var err error
	//	var connString  = "cockroachdb://root:@localhost:26257/dodo?sslmode=disable"
	//var connStringP  = "postgres://root:@localhost:26257/example?sslmode=disable"
	//var connStringdodo  = "postgres://root:@localhost:26257/dodo?sslmode=disable"

	var migrationDir = "file://internal/pkg/db/migrations/cockroach"

	InitStruct("dodo")

	driver ,_ := postgres.WithInstance(Db, &postgres.Config{})
	m,err:=migrate.NewWithDatabaseInstance(migrationDir, "postgres", driver)
	if err != nil {
		log.Println(err)
	}

	if err := m.Up(); err != nil {
		log.Println(err)
	}
	defer m.Close()

	log.Printf("UP")
	//db, err :=sql.Open("postgres",connString)
	//if err != nil {
	//	log.Fatal("error connecting to the database: ", err)
	//}

	//defer db.Close()

	if err:= Engine.Ping(); err != nil{
		log.Fatal(err)
	}

	defer Engine.Close()

	Engine.ShowSQL(true)
	Engine.Sync()
	err = Engine.Sync(new(User))
	if err != nil {
		log.Fatal("error connecting to the database: ", err)
	}

	if err := m.Down(); err != nil {
		log.Fatal(err)
	}
	log.Println("DOWN")


*/


}