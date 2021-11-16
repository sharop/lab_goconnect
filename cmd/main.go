package main

import (
	"fmt"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/cockroachdb"
	"log"
)

var settings = cockroachdb.ConnectionURL{
	Database: `booktown`,
	Host: `localhost`,
	User: `roach`,
	Password: `Q7gc8rEdS`,
}

type Book struct{
	ID			uint	`db:"id,omitempty"`
	TITLE		uint	`db:"title"`
	AuthorID	uint	`db:"author_id,omitempty"`
	SubjectID	uint	`db:"subject_id,omitempty"`

}

func (*Book) Store(sess db.Session) db.Store{
	return sess.Collection("books")
}

func main()  {
	sess, err := cockroachdb.Open(settings)
	if err!= nil{
		log.Fatalln(err)
	}
	defer sess.Close()

	exist, err:=sess.Collection("books").Exists()
	if err != nil{
		log.Fatal("Exist:", err)
	}
	if !exist{
		sess.Collection("books").Truncate()
	}
	var book Book
	err = sess.Get(&book, db.Cond{"id": 7808})
	if err != nil{
		log.Fatal("Find:", err)
	}
	fmt.Printf("Book: %#v", book)

}