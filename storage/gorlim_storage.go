package storage

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	GetGithubAuth(user string) (string, error)
	SaveGithubAuth(user string, auth string) error
}

type repoImpl struct {
	connection *sql.DB
}

func (r repoImpl) SaveGithubAuth(user string, auth string) error {
	stmt, err := r.connection.Prepare("insert or replace into github_auth (login, auth) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(user, auth)
	return err
}

func (r repoImpl) GetGithubAuth(user string) (string, error) {
	stmt, err := r.connection.Prepare("select auth from github_auth where login = ?")
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	rows, err := stmt.Query(user)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var auth string
		rows.Scan(&auth)
		return auth, nil
	}
	return "", errors.New("No auth found for " + user)
}

func Create(filename string) (*Storage, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	sqlStmt := `    create table if not exists github_auth
			(
				login text primary key,
				auth text
			);
			`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}
	var result Storage = Storage(repoImpl{connection: db})
	return &result, nil
}
