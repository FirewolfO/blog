package model

import "time"

const (
	PostDraft     = "draft"
	PostPublished = "published"
	PostArchived  = "archived"

	ReviewDraft    = "draft"
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
	ReviewCanceled = "canceled"

	ReviewSubmissionNew      = "new"
	ReviewSubmissionRevision = "revision"
)

type Post struct {
	ID                   string     `gorm:"primaryKey;size:40" json:"id"`
	Title                string     `gorm:"size:180;not null" json:"title"`
	Slug                 string     `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Excerpt              string     `gorm:"size:500" json:"excerpt"`
	Content              string     `gorm:"type:text;not null" json:"content"`
	CoverImageURL        string     `gorm:"size:500" json:"coverImageUrl"`
	Status               string     `gorm:"size:20;index;not null" json:"status"`
	CategoryID           string     `gorm:"size:40;index" json:"categoryId"`
	Category             *Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	TagsJSON             string     `gorm:"column:tags;type:text" json:"-"`
	AuthorID             string     `gorm:"size:80;index;not null" json:"authorId"`
	AuthorName           string     `gorm:"size:120;not null" json:"authorName"`
	ReviewStatus         string     `gorm:"size:20;index;not null;default:draft" json:"reviewStatus"`
	ReviewNote           string     `gorm:"size:500" json:"reviewNote,omitempty"`
	PendingRevisionID    string     `gorm:"size:40;index" json:"-"`
	PublishedAt          *time.Time `json:"publishedAt"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	CommentCount         int64      `gorm:"-" json:"commentCount"`
	RatingCount          int64      `gorm:"-" json:"ratingCount"`
	RatingAverage        float64    `gorm:"-" json:"ratingAverage"`
	MyRating             int        `gorm:"-" json:"myRating"`
	HasPendingChanges    bool       `gorm:"-" json:"hasPendingChanges"`
	CanEdit              bool       `gorm:"-" json:"canEdit"`
	CanDelete            bool       `gorm:"-" json:"canDelete"`
	RecommendationReason string     `gorm:"-" json:"recommendationReason,omitempty"`
	Tags                 []string   `gorm:"-" json:"tags"`
}

type PostRevision struct {
	ID            string     `gorm:"primaryKey;size:40" json:"id"`
	PostID        string     `gorm:"size:40;index;not null" json:"postId"`
	Title         string     `gorm:"size:180;not null" json:"title"`
	Slug          string     `gorm:"size:200;not null" json:"slug"`
	Excerpt       string     `gorm:"size:500" json:"excerpt"`
	Content       string     `gorm:"type:text;not null" json:"content"`
	CoverImageURL string     `gorm:"size:500" json:"coverImageUrl"`
	Status        string     `gorm:"size:20;not null" json:"status"`
	CategoryID    string     `gorm:"size:40;index" json:"categoryId"`
	TagsJSON      string     `gorm:"column:tags;type:text" json:"-"`
	SubmittedBy   string     `gorm:"size:80;index;not null" json:"submittedBy"`
	SubmittedName string     `gorm:"size:120;not null" json:"submittedName"`
	ReviewStatus  string     `gorm:"size:20;index;not null" json:"reviewStatus"`
	ReviewNote    string     `gorm:"size:500" json:"reviewNote,omitempty"`
	ReviewedBy    string     `gorm:"size:80" json:"reviewedBy,omitempty"`
	ReviewedAt    *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	Tags          []string   `gorm:"-" json:"tags"`
}

type ReviewSubmission struct {
	ID             string     `gorm:"primaryKey;size:40" json:"id"`
	PostID         string     `gorm:"size:40;index;not null" json:"postId"`
	RevisionID     string     `gorm:"size:40;index" json:"revisionId,omitempty"`
	Title          string     `gorm:"size:180;not null" json:"title"`
	SubmissionType string     `gorm:"size:20;index;not null" json:"submissionType"`
	SubmittedBy    string     `gorm:"size:80;index;not null" json:"submittedBy"`
	SubmittedName  string     `gorm:"size:120;not null" json:"submittedName"`
	ReviewStatus   string     `gorm:"size:20;index;not null" json:"reviewStatus"`
	ReviewNote     string     `gorm:"size:500" json:"reviewNote,omitempty"`
	ReviewedBy     string     `gorm:"size:80" json:"reviewedBy,omitempty"`
	ReviewedAt     *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt      time.Time  `gorm:"index" json:"submittedAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type ReviewNotification struct {
	ID                 string     `gorm:"primaryKey;size:40" json:"id"`
	ReviewSubmissionID string     `gorm:"size:40;uniqueIndex;not null" json:"reviewSubmissionId"`
	UserID             string     `gorm:"size:80;index;not null" json:"-"`
	PostID             string     `gorm:"size:40;index;not null" json:"postId"`
	Title              string     `gorm:"size:180;not null" json:"title"`
	ReviewStatus       string     `gorm:"size:20;index;not null" json:"reviewStatus"`
	ReviewNote         string     `gorm:"size:500" json:"reviewNote,omitempty"`
	CreatedAt          time.Time  `gorm:"index" json:"createdAt"`
	ReadAt             *time.Time `gorm:"index" json:"readAt,omitempty"`
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
	CanDelete  bool      `gorm:"-" json:"canDelete"`
}

type Rating struct {
	ID        string    `gorm:"primaryKey;size:40" json:"id"`
	PostID    string    `gorm:"size:40;uniqueIndex:idx_post_rater;not null" json:"postId"`
	RaterID   string    `gorm:"size:80;uniqueIndex:idx_post_rater;not null" json:"raterId"`
	Stars     int       `gorm:"not null" json:"stars"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
