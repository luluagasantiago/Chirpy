package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}
type User struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Password []byte `json:"password"`
}

type UserWithoutPass struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
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

func userAlreadyExits(email string, users map[int]User) bool {
	for _, val := range users {
		if val.Email == email {
			return true
		}
	}
	return false
}

func (db *DB) UserLookUp(email, password string) (UserWithoutPass, error) {
	userNoPass := UserWithoutPass{}

	cbUsers, err := db.loadDB()

	if err != nil {
		fmt.Print("Error when loading db")
		return UserWithoutPass{}, err
	}

	for _, user := range cbUsers.Users {
		if user.Email == email {
			err := bcrypt.CompareHashAndPassword(user.Password, []byte(password))
			if err == nil {
				userNoPass.Email = user.Email
				userNoPass.Id = user.Id
				return userNoPass, nil
			} else {
				return UserWithoutPass{}, errors.New("Passwords don't match")
			}
		}
	}
	return UserWithoutPass{}, errors.New("User not FOUND")

}

func (db *DB) CreateUser(email, password string) (UserWithoutPass, error) {
	user := User{}
	userNoPass := UserWithoutPass{}

	cbUsers, err := db.loadDB()

	if err != nil {
		fmt.Print("Error when loading db")
		return userNoPass, err
	}

	if userAlreadyExits(email, cbUsers.Users) {
		return userNoPass, errors.New("User Already exists")
	}

	// id will be equal to the actual number of Chirps. 0 base index
	id := len(cbUsers.Users) + 1
	user.Email = email
	user.Id = id
	encryptedPass, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		fmt.Println("Err when hashing password")
		return userNoPass, err
	}
	//func GenerateFromPassword(password []byte, cost int) ([]byte, error)
	user.Password = encryptedPass

	cbUsers.Users[id] = user

	err = db.writeDB(cbUsers)

	if err != nil {
		fmt.Print("Error when writing to db")
		return userNoPass, err
	}
	userNoPass.Email = email
	userNoPass.Id = id

	return userNoPass, nil
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
		Users:  map[int]User{},
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
	if len(dbStruct.Users) == 0 {
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
