package db

import (
	"database/sql"
	"fmt"
	"infra-management/internal/config"

	_ "github.com/lib/pq"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	// 尝试设置客户端编码，如果失败则忽略
	_, err = db.Exec("SET client_encoding TO 'UTF8'")
	if err != nil {
		fmt.Println("警告: 设置客户端编码失败: ", err)
	}

	return db, nil
}
