package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
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
			`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}
	var result Storage = Storage(repoImpl{connection: db, noRepos: []*Repo{}})
	return &result, nil
}
