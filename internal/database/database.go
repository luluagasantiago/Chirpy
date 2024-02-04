package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	f, err := os.Create(path)
	db := DB{}
	if err != nil {
		return nil, err
	}
	db.path = f.Name()
	db.mux = &sync.RWMutex{}
	return &db, nil
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(body string) (Chirp, error) {
	chirp := Chirp{}

	cbChirps, err := db.loadDB()

	if err != nil {
		fmt.Print("Error when loading db")
		return chirp, err
	}
	// id will be equal to the actual number of Chirps. 0 base index
	id := len(cbChirps.Chirps) + 1
	chirp.Body = body
	chirp.Id = id
	cbChirps.Chirps[id] = chirp

	err = db.writeDB(cbChirps)

	if err != nil {
		fmt.Print("Error when writing to db")
		return chirp, err
	}

	return chirp, nil
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	dbStruct, err := db.loadDB()

	if err != nil {
		return nil, err
	}

	chirps := []Chirp{}
	for _, val := range dbStruct.Chirps {
		chirps = append(chirps, val)
	}
	return chirps, nil
}

// ensureDB creates a new database file if it doesn't exist
//func (db *DB) ensureDB() error

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	db.mux.RLock()
	defer db.mux.RUnlock()
	//jsonFile, err := os.Open(db.path)
	dbStruct := DBStructure{
		Chirps: map[int]Chirp{},
	}
	/*if err != nil {
		fmt.Print("\nError when open db.path\n")
		return dbStruct, err
	}*/
	//defer jsonFile.Close()
	//byteValue, _ := io.ReadAll(jsonFile)
	byteValue, err := os.ReadFile(db.path)
	if errors.Is(err, os.ErrNotExist) {
		return dbStruct, err
	}
	if err != nil {
		fmt.Print("\nError when Reading File\n")
		return dbStruct, err
	}

	err = json.Unmarshal(byteValue, &dbStruct)

	if len(dbStruct.Chirps) == 0 {
		return dbStruct, nil
	}
	if err != nil {
		fmt.Print("\nError when Unmarshalling json\n")
		return dbStruct, err
	}
	return dbStruct, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	jsonFile, err := os.Open(db.path)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	jsonData, err := json.Marshal(dbStructure)

	os.WriteFile(db.path, jsonData, 0666)

	if err != nil {
		return err
	}

	return nil

}
