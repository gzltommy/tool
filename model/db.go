package model

import (
	"fmt"
	"sync"
	"time"
	"tool-attendance/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once sync.Once
	db   *gorm.DB
)

func Init(cfg *config.MysqlConfig) error {
	var err error
	once.Do(
		func() {
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName)
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Info),
				//NamingStrategy: schema.NamingStrategy{
				//	//TablePrefix:   "",
				//	SingularTable: true,
				//	//NameReplacer:  nil,
				//	//NoLowerCase:   false,
				//},
			})
			if err != nil {
				return
			}
			idb, _ := db.DB()
			idb.SetConnMaxIdleTime(120 * time.Second)
			idb.SetConnMaxLifetime(7200 * time.Second)
			idb.SetMaxOpenConns(200)
			idb.SetMaxIdleConns(10)

			if err := idb.Ping(); err != nil {
				return
			}
		})

	return nil
}

func GetDb() *gorm.DB {
	return db
}

func GetTableName(v interface{}) string {
	stat := gorm.Statement{DB: DB()}
	err := stat.Parse(v)
	if err != nil {
		return ""
	}
	return stat.Schema.Table
}

func DB() *gorm.DB {
	return db
}
