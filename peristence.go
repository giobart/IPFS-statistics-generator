package main

import (
	"encoding/json"
	"github.com/mitchellh/mapstructure"
	"github.com/nanobox-io/golang-scribble"
)

const (
	DWrite     = "write"
	DRead      = "read"
	DReadAll   = "readall"
	DBPosition = ".database"
)

//struct representing the query to the database
type Query struct {
	operation  string
	collection string
	key        string
	object     interface{}
}

//struct representing the database, must be a singleton
type Database struct {
	db     *scribble.Driver
	result chan interface{}
	query  chan Query
	close  chan bool
}

func (db *Database) dbInit() {

	db.result = make(chan interface{}, 1)
	db.query = make(chan Query, 1)
	db.close = make(chan bool, 1)

	go func(db *Database) {
		//database instantiation
		currdb, err := scribble.New(DBPosition, nil)
		if err != nil {
			log.Error(err)
			panic(1)
		}
		db.db = currdb

		//iterating waiting for query
		for {
			select {
			case q := <-db.query:
				switch q.operation {
				//persist this object to the given collection
				case DWrite:
					if err := db.db.Write(q.collection, q.key, q.object); err != nil {
						log.Error(err)
					}
				//read from persistence and send result with db.result channel
				case DRead:
					var object interface{}

					if err := db.db.Read(q.collection, q.key, &object); err != nil {
						log.Error(err)
						db.result <- nil
					} else {
						db.result <- object
					}
				//ead all the values in a collection
				case DReadAll:
					records, err := db.db.ReadAll(q.collection)
					if err != nil {
						log.Error(err)
						db.result <- nil
					} else {
						list := make([]map[string]interface{}, 1)

						for _, f := range records {
							var atom map[string]interface{}

							if err := json.Unmarshal([]byte(f), &atom); err != nil {
								log.Error(err)
							} else {
								list = append(list, atom)
							}
						}
						db.result <- list
					}
				}
			case <-db.close:
				return
			}
		}
	}(db)

}

func (db *Database) dbClose() {
	db.close <- true
}

func (db *Database) dbRead(collection string, key string) map[string]interface{} {
	q := Query{
		operation:  DRead,
		collection: collection,
		key:        key,
		object:     nil,
	}

	db.query <- q

	if res := <-db.result; res == nil {
		return nil
	} else {
		return res.(map[string]interface{})
	}
}

func (db *Database) dbWrite(collection string, key string, object interface{}) {
	q := Query{
		operation:  DWrite,
		collection: collection,
		key:        key,
		object:     object,
	}

	db.query <- q

}

func (db *Database) dbReadAll(collection string) []map[string]interface{} {
	q := Query{
		operation:  DReadAll,
		collection: collection,
		key:        "",
		object:     nil,
	}

	db.query <- q

	if res := <-db.result; res == nil {
		return nil
	} else {
		return res.([]map[string]interface{})
	}

}

func test_database() {

	var database = Database{}
	database.dbInit()

	//write into the db
	peer := Peer{
		Addr:    "127.0.0.1",
		Cid:     "asfasddfads",
		Latency: "Infinite",
	}

	database.dbWrite("peers", "1", peer)
	database.dbWrite("peers", "2", peer)

	resultpeer1 := Peer{}
	result1 := database.dbRead("peers", "1")
	err := mapstructure.Decode(result1, &resultpeer1)
	if err != nil {
		log.Error(err)
	}

	resultpeer2 := Peer{}
	result2 := database.dbRead("peers", "5")
	err = mapstructure.Decode(result2, &resultpeer2)
	if err != nil {
		log.Error(err)
	}

	if resultpeer1 == peer {
		print("Succeded \n")
	} else {
		print("Failed \n")
	}

	if resultpeer2 == peer {
		print("Failed \n")
	} else {
		print("Succeded \n")
	}

	list := database.dbReadAll("peers")
	peerList := make([]Peer, 0)
	for _, el := range list {
		if el != nil {
			peer := Peer{}
			if err := mapstructure.Decode(el, &peer); err != nil {
				log.Error(err)
			} else {
				peerList = append(peerList, peer)
			}
		}
	}
	print(len(peerList))

}
