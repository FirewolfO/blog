package store

import (
	"fmt"

	"blog/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Store struct{ DB *gorm.DB }

func Open(dsn string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open blog database: %w", err)
	}
	if err := db.AutoMigrate(&model.Post{}, &model.PostRevision{}, &model.ReviewSubmission{}, &model.ReviewNotification{}, &model.Category{}, &model.Comment{}, &model.Rating{}, &model.Media{}, &model.Session{}, &model.OAuthState{}); err != nil {
		return nil, fmt.Errorf("migrate blog database: %w", err)
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	db, err := s.DB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}
