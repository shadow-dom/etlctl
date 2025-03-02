package etl

import (
	"database/sql"
	"fmt"
	"os/user"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"
)

type DBStorage struct {
	Name       string            `yaml:"name"`
	Type       string            `yaml:"type"`
	Connection map[string]string `yaml:"connection"`
	Query      string            `yaml:"query"`
}

func expandPath(path string) (string, error) {
	if path[:2] == "~/" {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return path, nil
}

func (dbs *DBStorage) getDSN() (string, error) {
	switch dbs.Type {
	case "sqlserver":
		user, ok := dbs.Connection["user"]

		if ok {
			return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s",
				user, dbs.Connection["password"],
				dbs.Connection["host"], dbs.Connection["port"], dbs.Connection["database"]), nil
		}

		return fmt.Sprintf("sqlserver://@%s:%s?database=%s&trusted_connection=yes",
			dbs.Connection["host"], dbs.Connection["port"], dbs.Connection["database"]), nil
	case "sqlite3":
		path, err := expandPath(dbs.Connection["filepath"])

		if err != nil {
			fmt.Println("Error expanding file path:", err)
			return "", err
		}

		return path, nil
	default:
		return "", fmt.Errorf("unsupported database type: %s", dbs.Type)
	}
}

func (dbs *DBStorage) Connect() (*sql.DB, error) {
	dsn, err := dbs.getDSN()

	if err != nil {
		return nil, err
	}

	db, err := sql.Open(dbs.Type, dsn)

	if err != nil {
		return nil, err
	}

	return db, nil
}
