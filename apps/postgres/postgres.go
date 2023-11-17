package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mirzakhany/sysd"
)

var _ sysd.App = &Postgres{}

type Postgres struct {
	DatabaseName string
	Username     string
	Password     string
	Host         string
	Port         int

	conn *pgxpool.Pool
}

func (p *Postgres) New(DatabaseName, Username, Password, Host string, Port int) *Postgres {
	return &Postgres{
		DatabaseName: DatabaseName,
		Username:     Username,
		Password:     Password,
		Host:         Host,
		Port:         Port,
	}
}

func NewWithURI(uri string) (*Postgres, error) {
	conn, err := pgx.ParseConfig(uri)
	if err != nil {
		return nil, err
	}

	return &Postgres{
		DatabaseName: conn.Database,
		Username:     conn.User,
		Password:     conn.Password,
		Host:         conn.Host,
		Port:         int(conn.Port),
	}, nil
}

func (p *Postgres) Start(ctx context.Context) error {
	conn, err := pgxpool.NewWithConfig(ctx, &pgxpool.Config{
		ConnConfig: &pgx.ConnConfig{
			Config: pgconn.Config{
				Host:     p.Host,
				Port:     uint16(p.Port),
				Database: p.DatabaseName,
				User:     p.Username,
				Password: p.Password,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}
	p.conn = conn

	return sysd.ShutdownGracefully(ctx, func() error {
		conn.Close()
		return nil
	})
}

func (p *Postgres) Status(ctx context.Context) error {
	return p.conn.Ping(ctx)
}

func (p *Postgres) Name() string {
	return "postgres"
}

func (p *Postgres) Connection() (*pgxpool.Pool, error) {
	if p.conn != nil {
		return p.conn, nil
	}
	return nil, fmt.Errorf("postgres connection is nil")
}
