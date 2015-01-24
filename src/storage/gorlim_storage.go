package storage

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	GetGithubAuth(user string) (string, error)
	SaveGithubAuth(user, auth string) error
	GetRepos(needle string) ([]*Repo, error)
	AddRepo(myType, origin, target string) error
}

type Repo struct {
	Type   *string
	Origin *string
	Target *string
}

type repoImpl struct {
	connection *sql.DB
	noRepos    []*Repo
}

func (r repoImpl) SaveGithubAuth(user, auth string) error {
	stmt, err := r.connection.Prepare("insert or replace into github_auth (login, auth) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Query(user, auth)
	return err
}

func (r repoImpl) GetGithubAuth(user string) (string, error) {
	stmt, err := r.connection.Prepare("select auth from github_auth where login equals ?")
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

func (r repoImpl) AddRepo(myType, origin, target string) error {
	stmt, err := r.connection.Prepare("insert into repositories (type, origin, target) values(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(myType, origin, target)
	return err
}

func (r repoImpl) GetRepos(needle string) ([]*Repo, error) {
	stmt, err := r.connection.Prepare("select type, origin, target from repositories where id like ?")
	if err != nil {
		return r.noRepos, err
	}
	defer stmt.Close()
	rows, err := stmt.Query("%" + needle + "%")
	if err != nil {
		return r.noRepos, err
	}
	defer rows.Close()
	result := make([]*Repo, 0, 20)
	for rows.Next() {
		var myType string
		var origin string
		var target string
		rows.Scan(&myType, &origin, &target)
		result = append(result, &Repo{Origin: &origin, Target: &target, Type: &myType})
	}
	return result, nil
}

func Create(filename string) (*Storage, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	sqlStmt := `
		  create table if not exists repositories 
			(
				id integer not null primary key autoincrement, 
				type text not null, 
				origin text not null, 
				target text not null
			) ;
			create table if not exists github_auth
			(
				login text primary key,
				auth text
			);
			`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}
	var result Storage = Storage(repoImpl{connection: db, noRepos: []*Repo{}})
	return &result, nil
}
