package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type RefreshToken struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

type User struct {
	ID           int          `json:"id"`
	Email        string       `json:"email"`
	Password     []byte       `json:"password"`
	RefreshToken RefreshToken `json:"refresh_token"`
}
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
	Users  map[int]User  `json:"users"`
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
	dbStructure := DBStructure{
		Chirps: make(map[int]Chirp),
		Users:  make(map[int]User),
	}
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
	fmt.Printf("Marshaled JSON data: %s\n", string(dbdata))
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
	dbs, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}
	emptyChirp := Chirp{}
	if dbs.Chirps[id] == emptyChirp {
		return emptyChirp, errors.New("not found")
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

func (db *DB) GetUsers() ([]User, error) {
	users := make([]User, 0)
	dbs, err := db.loadDB()
	if err != nil {
		return users, err
	}
	for _, n := range dbs.Users {
		users = append(users, n)
	}

	return users, nil
}

func (db *DB) GetUser(id int) (User, error) {
	emptyUser := User{}
	dbs, err := db.loadDB()
	if err != nil {
		return emptyUser, err
	}
	if len(dbs.Users[id].Email) == 0 {
		return emptyUser, errors.New("not found")
	}
	return dbs.Users[id], nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	emptyUser := User{}
	dbs, err := db.loadDB()
	if err != nil {
		return emptyUser, err
	}
	for i := 1; i <= len(dbs.Users); i++ {
		if dbs.Users[i].Email == email {
			return dbs.Users[i], nil
		}
	}
	return emptyUser, errors.New("User not found")
}

func (db *DB) GetUserByRefreshToken(refreshToken string) (User, error) {
	emptyUser := User{}
	dbs, err := db.loadDB()
	if err != nil {
		return emptyUser, err
	}
	for i := 1; i <= len(dbs.Users); i++ {
		if dbs.Users[i].RefreshToken.Token == refreshToken {
			if dbs.Users[i].RefreshToken.Expiry.Before(time.Now()) {
				return emptyUser, errors.New("Token has expired")
			} else {
				return dbs.Users[i], nil
			}
		}
	}
	return emptyUser, errors.New("User matching token not found")
}

func (db *DB) CreateUser(email string, password []byte) (User, error) {
	users, err := db.GetUsers()
	if err != nil {
		return User{}, err
	}
	newID := len(users) + 1
	newUser := User{
		ID:       newID,
		Email:    email,
		Password: password,
	}
	structure, err := db.loadDB()
	structure.Users[newID] = newUser
	db.writeDB(structure)
	fmt.Printf("Added user id %v: %s\n", newID, email)
	return newUser, nil
}

func (db *DB) UpdateUser(id int, email string, password []byte) (User, error) {
	structure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}
	structure.Users[id] = User{
		ID:           id,
		Email:        email,
		Password:     password,
		RefreshToken: structure.Users[id].RefreshToken,
	}
	db.writeDB(structure)
	fmt.Printf("Updated user %v: %s\n", id, email)
	return db.GetUser(id)
}
func (db *DB) AddRefreshToken(id int, refreshToken string) (User, error) {
	structure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	expiryDate := time.Now().Add(time.Hour * 24 * 60)

	user := structure.Users[id]
	user.RefreshToken.Token = refreshToken
	user.RefreshToken.Expiry = expiryDate
	structure.Users[id] = user
	err = db.writeDB(structure)
	if err != nil {
		return User{}, err
	}
	return db.GetUser(id)
}
func (db *DB) RevokeRefreshToken(id int) error {
	structure, err := db.loadDB()
	if err != nil {
		return err
	}

	user := structure.Users[id]
	user.RefreshToken = RefreshToken{}
	structure.Users[id] = user
	err = db.writeDB(structure)
	if err != nil {
		return err
	}
	return nil
}
