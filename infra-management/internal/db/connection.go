package db

import (
	"database/sql"
	"fmt"
	"infra-management/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable client_encoding=utf8",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	// 在连接字符串中设置客户端编码为 UTF-8
	_, err = db.Exec("SET client_encoding TO 'UTF8'")
	if err != nil {
		return nil, err
	}

	return db, nil
}
