package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"errors"
)

type Chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

func NewDB(path string) (*DB, error) {
	db := DB{
		path: path,
		mux:  &sync.RWMutex{},
	}
	db.ensureDB()
	db.loadDB()
	return &db, nil
}

func (db *DB) ensureDB() error {
	db.mux.Lock()
	defer db.mux.Unlock()
	_, err := os.ReadFile(db.path)
	if err != nil {
		os.WriteFile(db.path, nil, 0644)
	}
	return err
}

func (db *DB) loadDB() (DBStructure, error) {
	dbStructure := DBStructure{Chirps: make(map[int]Chirp)}
	txt, err := os.ReadFile(db.path)
	err = json.Unmarshal(txt, &dbStructure)
	if err != nil {
		err = db.writeDB(dbStructure)
	}
	return dbStructure, err
}

func (db *DB) writeDB(dbstructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbdata, err := json.MarshalIndent(dbstructure, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(db.path, dbdata, 644)
	if err != nil {
		return err
	}
	fmt.Printf("wrote database to %s\n", db.path)
	return nil
}

func (db *DB) GetChirps() ([]Chirp, error) {
	chirps := make([]Chirp, 0)
	dbs, err := db.loadDB()
	if err != nil {
		return chirps, err
	}
	for _, n := range dbs.Chirps {
		chirps = append(chirps, n)
	}

	return chirps, nil
}

func (db *DB) GetChirp(id int) (Chirp, error) {
    dbs, err:=db.loadDB()
    if err != nil {
	return Chirp{}, err
    }
    emptyChirp:=Chirp{}
    if dbs.Chirps[id] == emptyChirp {
	return emptyChirp, errors.New("not found" )
    }
    return dbs.Chirps[id], nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	chirps, err := db.GetChirps()
	if err != nil {
		return Chirp{}, err
	}
	newID := len(chirps) + 1
	newChirp := Chirp{
		ID:   newID,
		Body: body,
	}
	structure, err := db.loadDB()
	structure.Chirps[newID] = newChirp
	db.writeDB(structure)
	fmt.Printf("Added chirp id %v: %s\n", newID, body)
	return newChirp, nil
}
