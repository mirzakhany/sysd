package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mirzakhany/sysd"
)

var _ sysd.App = &Postgres{}

type Postgres struct {
	connConf *pgxpool.Config
	conn     *pgxpool.Pool
}

func New(DatabaseName, Username, Password, Host string, Port int) (*Postgres, error) {
	uri := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", Username, Password, Host, Port, DatabaseName)
	conf, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, err
	}

	return &Postgres{connConf: conf}, nil
}

func NewWithURI(uri string) (*Postgres, error) {
	conf, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, err
	}

	return &Postgres{connConf: conf}, nil
}

func (p *Postgres) Start(ctx context.Context) error {
	conn, err := pgxpool.NewWithConfig(ctx, p.connConf)
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
