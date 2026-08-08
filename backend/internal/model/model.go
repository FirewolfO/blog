package model

import "time"

const (
	PostDraft     = "draft"
	PostPublished = "published"
	PostArchived  = "archived"
)

type Post struct {
	ID            string     `gorm:"primaryKey;size:40" json:"id"`
	Title         string     `gorm:"size:180;not null" json:"title"`
	Slug          string     `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Excerpt       string     `gorm:"size:500" json:"excerpt"`
	Content       string     `gorm:"type:text;not null" json:"content"`
	CoverImageURL string     `gorm:"size:500" json:"coverImageUrl"`
	Status        string     `gorm:"size:20;index;not null" json:"status"`
	CategoryID    string     `gorm:"size:40;index" json:"categoryId"`
	Category      *Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	TagsJSON      string     `gorm:"column:tags;type:text" json:"-"`
	AuthorID      string     `gorm:"size:80;index;not null" json:"authorId"`
	AuthorName    string     `gorm:"size:120;not null" json:"authorName"`
	PublishedAt   *time.Time `json:"publishedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	CommentCount  int64      `gorm:"-" json:"commentCount"`
	Tags          []string   `gorm:"-" json:"tags"`
}

type Category struct {
	ID          string    `gorm:"primaryKey;size:40" json:"id"`
	Name        string    `gorm:"size:80;uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Description string    `gorm:"size:300" json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	PostCount   int64     `gorm:"-" json:"postCount"`
}

type Comment struct {
	ID         string    `gorm:"primaryKey;size:40" json:"id"`
	PostID     string    `gorm:"size:40;index;not null" json:"postId"`
	Content    string    `gorm:"size:1000;not null" json:"content"`
	AuthorID   string    `gorm:"size:80;index;not null" json:"authorId"`
	AuthorName string    `gorm:"size:120;not null" json:"authorName"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Media struct {
	ID          string    `gorm:"primaryKey;size:40" json:"id"`
	ObjectKey   string    `gorm:"size:500;uniqueIndex;not null" json:"objectKey"`
	Filename    string    `gorm:"size:255;not null" json:"filename"`
	ContentType string    `gorm:"size:120;not null" json:"contentType"`
	Size        int64     `json:"size"`
	URL         string    `gorm:"size:500;not null" json:"url"`
	UploaderID  string    `gorm:"size:80;index;not null" json:"uploaderId"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Session struct {
	TokenHash   string    `gorm:"primaryKey;size:64"`
	UserID      string    `gorm:"size:80;index;not null"`
	Username    string    `gorm:"size:100;not null"`
	DisplayName string    `gorm:"size:120;not null"`
	ExpiresAt   time.Time `gorm:"index;not null"`
	CreatedAt   time.Time
}

type OAuthState struct {
	StateHash   string    `gorm:"primaryKey;size:64"`
	RedirectURI string    `gorm:"size:500;not null"`
	ExpiresAt   time.Time `gorm:"index;not null"`
	CreatedAt   time.Time
}
