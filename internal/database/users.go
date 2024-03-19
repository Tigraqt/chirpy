package database

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var ErrAlreadyExists = errors.New("already exists")

type User struct {
	ID             int    `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
	IsChirpyRed    bool   `json:"is_chirpy_red"`
}

func userExists(email string, users map[int]User) *User {
	for _, user := range users {
		if user.Email == email {
			return &user
		}
	}
	return nil
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return nil, err
	}

	for _, user := range dbStructure.Users {
		if user.Email == email {
			return &user, nil
		}
	}

	return nil, ErrNotExist
}

func (db *DB) CreateUser(email, hashedPassword string) (*User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return nil, err
	}

	if user := userExists(email, dbStructure.Users); user != nil {
		return nil, fmt.Errorf("user with Email %s is already exists", email)
	}

	id := len(dbStructure.Users) + 1
	user := User{
		ID:             id,
		Email:          email,
		HashedPassword: hashedPassword,
	}
	dbStructure.Users[id] = user

	err = db.writeDB(dbStructure)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *DB) GetUser(email, password string) (*User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return nil, err
	}

	if user := userExists(email, dbStructure.Users); user == nil {
		return nil, fmt.Errorf("user with Email %s not found", email)
	} else {
		err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
		if err == nil {
			return user, nil
		}

		return nil, err
	}
}

func (db *DB) UpdateUser(id int, email, hashedPassword string) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	user, ok := dbStructure.Users[id]
	if !ok {
		return User{}, ErrNotExist
	}

	user.Email = email
	user.HashedPassword = hashedPassword
	dbStructure.Users[id] = user

	err = db.writeDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (db *DB) UpgradeChirpyRed(
	id int,
) (User, error) {
	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	user, ok := dbStructure.Users[id]
	if !ok {
		return User{}, ErrNotExist
	}

	user.IsChirpyRed = true
	dbStructure.Users[id] = user

	err = db.writeDB(dbStructure)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
