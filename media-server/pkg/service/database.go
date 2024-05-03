package service

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"go.uber.org/fx"
)

type DatabaseConfig struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
	Driver   string
}

func (dconf *DatabaseConfig) GetURI() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		dconf.Driver,
		dconf.Username,
		dconf.Password,
		dconf.Host,
		dconf.Port,
		dconf.Database,
	)
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:   "postgres",
		Username: "admin",
		Password: "admin",
		Host:     "postgres",
		Port:     "5432",
		Database: "postgres",
	}
}

type NewDatabaseConnectionParams struct {
	fx.In
	Lifecycle fx.Lifecycle

	Config *DatabaseConfig
}

func NewDatabaseConnection(params NewDatabaseConnectionParams) (*sql.DB, error) {
	conn, err := sql.Open(params.Config.Driver, params.Config.GetURI()+"?sslmode=disable")
	if err != nil {
		return nil, err
	}
	params.Lifecycle.Append(fx.StopHook(conn.Close))
	return conn, nil
}

var DatabaseModule = fx.Module("db",
	fx.Provide(
		NewDatabaseConfig,
		NewDatabaseConnection,
	),
	fx.Invoke(func(conn *sql.DB) {
		log.Println("conn: ", conn)
		log.Printf("%+v", conn.Stats())
	}),
)
