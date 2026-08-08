package blog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"blog/internal/model"
	"blog/internal/store"
	"gorm.io/gorm"
)

var (
	ErrInvalid  = errors.New("invalid input")
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	nonSlug     = regexp.MustCompile(`[^a-z0-9]+`)
)

type Identity struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Source      string   `json:"source,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func (i Identity) Can(permission string) bool {
	if i.Source != "permission" {
		return true
	}
	for _, value := range i.Permissions {
		if value == "*" || value == permission {
			return true
		}
	}
	return false
}

type PostInput struct {
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	Excerpt       string   `json:"excerpt"`
	Content       string   `json:"content"`
	CoverImageURL string   `json:"coverImageUrl"`
	Status        string   `json:"status"`
	CategoryID    string   `json:"categoryId"`
	Tags          []string `json:"tags"`
}

type CategoryInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type PostPage struct {
	Items    []model.Post `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type Dashboard struct {
	Posts      int64        `json:"posts"`
	Published  int64        `json:"published"`
	Drafts     int64        `json:"drafts"`
	Comments   int64        `json:"comments"`
	Categories int64        `json:"categories"`
	Recent     []model.Post `json:"recent"`
}

type Service struct{ store *store.Store }

func New(database *store.Store) *Service { return &Service{store: database} }

func (s *Service) Dashboard() (*Dashboard, error) {
	result := &Dashboard{Recent: []model.Post{}}
	for target, query := range map[*int64]any{
		&result.Posts: &model.Post{}, &result.Comments: &model.Comment{}, &result.Categories: &model.Category{},
	} {
		if err := s.store.DB.Model(query).Count(target).Error; err != nil {
			return nil, err
		}
	}
	if err := s.store.DB.Model(&model.Post{}).Where("status = ?", model.PostPublished).Count(&result.Published).Error; err != nil {
		return nil, err
	}
	if err := s.store.DB.Model(&model.Post{}).Where("status = ?", model.PostDraft).Count(&result.Drafts).Error; err != nil {
		return nil, err
	}
	if err := s.store.DB.Order("updated_at DESC").Limit(5).Find(&result.Recent).Error; err != nil {
		return nil, err
	}
	decodePostTags(result.Recent)
	return result, nil
}

func (s *Service) ListPosts(search, status, categoryID string, page, pageSize int) (*PostPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.store.DB.Model(&model.Post{}).Preload("Category")
	if value := strings.TrimSpace(search); value != "" {
		like := "%" + value + "%"
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR content LIKE ?", like, like, like)
	}
	if value := strings.TrimSpace(status); value != "" {
		query = query.Where("status = ?", value)
	}
	if value := strings.TrimSpace(categoryID); value != "" {
		query = query.Where("category_id = ?", value)
	}
	result := &PostPage{Items: []model.Post{}, Page: page, PageSize: pageSize}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&result.Items).Error; err != nil {
		return nil, err
	}
	for index := range result.Items {
		_ = s.store.DB.Model(&model.Comment{}).Where("post_id = ?", result.Items[index].ID).Count(&result.Items[index].CommentCount).Error
	}
	decodePostTags(result.Items)
	return result, nil
}

func (s *Service) GetPost(id string) (*model.Post, error) {
	var post model.Post
	if err := s.store.DB.Preload("Category").Where("id = ?", id).First(&post).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	decodePostTags([]model.Post{post})
	_ = json.Unmarshal([]byte(post.TagsJSON), &post.Tags)
	_ = s.store.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&post.CommentCount).Error
	return &post, nil
}

func (s *Service) CreatePost(actor Identity, input PostInput) (*model.Post, error) {
	post, err := normalizePost(nil, actor, input)
	if err != nil {
		return nil, err
	}
	post.ID = newID("post")
	if err := s.ensureSlug(&post.Slug, ""); err != nil {
		return nil, err
	}
	if err := s.validateCategory(post.CategoryID); err != nil {
		return nil, err
	}
	if err := s.store.DB.Create(post).Error; err != nil {
		return nil, err
	}
	return s.GetPost(post.ID)
}

func (s *Service) UpdatePost(id string, actor Identity, input PostInput) (*model.Post, error) {
	current, err := s.GetPost(id)
	if err != nil {
		return nil, err
	}
	updated, err := normalizePost(current, actor, input)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSlug(&updated.Slug, current.ID); err != nil {
		return nil, err
	}
	if err := s.validateCategory(updated.CategoryID); err != nil {
		return nil, err
	}
	if err := s.store.DB.Model(&model.Post{}).Where("id = ?", id).Updates(map[string]any{
		"title": updated.Title, "slug": updated.Slug, "excerpt": updated.Excerpt, "content": updated.Content,
		"cover_image_url": updated.CoverImageURL, "status": updated.Status, "category_id": updated.CategoryID,
		"tags": updated.TagsJSON, "published_at": updated.PublishedAt,
	}).Error; err != nil {
		return nil, err
	}
	return s.GetPost(id)
}

func (s *Service) PublishPost(id string) (*model.Post, error) {
	if _, err := s.GetPost(id); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.store.DB.Model(&model.Post{}).Where("id = ?", id).Updates(map[string]any{"status": model.PostPublished, "published_at": &now}).Error; err != nil {
		return nil, err
	}
	return s.GetPost(id)
}

func (s *Service) DeletePost(id string) error {
	if _, err := s.GetPost(id); err != nil {
		return err
	}
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", id).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.Post{}).Error
	})
}

func (s *Service) ListCategories() ([]model.Category, error) {
	result := []model.Category{}
	if err := s.store.DB.Order("name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	for index := range result {
		_ = s.store.DB.Model(&model.Post{}).Where("category_id = ?", result[index].ID).Count(&result[index].PostCount).Error
	}
	return result, nil
}

func (s *Service) CreateCategory(input CategoryInput) (*model.Category, error) {
	category, err := normalizeCategory(input)
	if err != nil {
		return nil, err
	}
	category.ID = newID("cat")
	if err := s.ensureCategoryUnique(category, ""); err != nil {
		return nil, err
	}
	if err := s.store.DB.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (s *Service) UpdateCategory(id string, input CategoryInput) (*model.Category, error) {
	var current model.Category
	if err := s.store.DB.Where("id = ?", id).First(&current).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	category, err := normalizeCategory(input)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCategoryUnique(category, id); err != nil {
		return nil, err
	}
	if err := s.store.DB.Model(&current).Updates(map[string]any{"name": category.Name, "slug": category.Slug, "description": category.Description}).Error; err != nil {
		return nil, err
	}
	return &current, s.store.DB.Where("id = ?", id).First(&current).Error
}

func (s *Service) DeleteCategory(id string) error {
	var count int64
	if err := s.store.DB.Model(&model.Category{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	if err := s.store.DB.Model(&model.Post{}).Where("category_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: category is in use", ErrConflict)
	}
	return s.store.DB.Where("id = ?", id).Delete(&model.Category{}).Error
}

func (s *Service) ListComments(postID string) ([]model.Comment, error) {
	if _, err := s.GetPost(postID); err != nil {
		return nil, err
	}
	result := []model.Comment{}
	return result, s.store.DB.Where("post_id = ?", postID).Order("created_at DESC").Find(&result).Error
}

func (s *Service) CreateComment(postID string, actor Identity, content string) (*model.Comment, error) {
	if _, err := s.GetPost(postID); err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 1000 {
		return nil, fmt.Errorf("%w: comment content is required and limited to 1000 characters", ErrInvalid)
	}
	comment := &model.Comment{ID: newID("comment"), PostID: postID, Content: content, AuthorID: actor.ID, AuthorName: identityName(actor)}
	return comment, s.store.DB.Create(comment).Error
}

func (s *Service) DeleteComment(id string) error {
	result := s.store.DB.Where("id = ?", id).Delete(&model.Comment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) SaveMedia(media *model.Media) error { return s.store.DB.Create(media).Error }

func (s *Service) GetMedia(id string) (*model.Media, error) {
	var media model.Media
	if err := s.store.DB.Where("id = ?", id).First(&media).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &media, nil
}

func normalizePost(current *model.Post, actor Identity, input PostInput) (*model.Post, error) {
	input.Title, input.Content = strings.TrimSpace(input.Title), strings.TrimSpace(input.Content)
	if input.Title == "" || len([]rune(input.Title)) > 180 || input.Content == "" {
		return nil, fmt.Errorf("%w: title and content are required", ErrInvalid)
	}
	if input.Status == "" {
		input.Status = model.PostDraft
	}
	if input.Status != model.PostDraft && input.Status != model.PostPublished && input.Status != model.PostArchived {
		return nil, fmt.Errorf("%w: invalid post status", ErrInvalid)
	}
	slug := slugify(input.Slug)
	if slug == "" {
		slug = slugify(input.Title)
	}
	if slug == "" {
		slug = "post-" + strings.ToLower(newID("")[0:8])
	}
	tags, _ := json.Marshal(uniqueTags(input.Tags))
	post := &model.Post{Title: input.Title, Slug: slug, Excerpt: strings.TrimSpace(input.Excerpt), Content: input.Content, CoverImageURL: strings.TrimSpace(input.CoverImageURL), Status: input.Status, CategoryID: strings.TrimSpace(input.CategoryID), TagsJSON: string(tags), AuthorID: actor.ID, AuthorName: identityName(actor)}
	if current != nil {
		post.ID, post.AuthorID, post.AuthorName, post.PublishedAt = current.ID, current.AuthorID, current.AuthorName, current.PublishedAt
	}
	if input.Status == model.PostPublished && post.PublishedAt == nil {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	return post, nil
}

func normalizeCategory(input CategoryInput) (*model.Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 80 {
		return nil, fmt.Errorf("%w: category name is required", ErrInvalid)
	}
	slug := slugify(input.Slug)
	if slug == "" {
		slug = slugify(input.Name)
	}
	if slug == "" {
		return nil, fmt.Errorf("%w: category slug is required", ErrInvalid)
	}
	return &model.Category{Name: input.Name, Slug: slug, Description: strings.TrimSpace(input.Description)}, nil
}

func (s *Service) ensureSlug(slug *string, exceptID string) error {
	var count int64
	query := s.store.DB.Model(&model.Post{}).Where("slug = ?", *slug)
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: post slug already exists", ErrConflict)
	}
	return nil
}

func (s *Service) ensureCategoryUnique(category *model.Category, exceptID string) error {
	var count int64
	query := s.store.DB.Model(&model.Category{}).Where("name = ? OR slug = ?", category.Name, category.Slug)
	if exceptID != "" {
		query = query.Where("id <> ?", exceptID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: category name or slug already exists", ErrConflict)
	}
	return nil
}

func (s *Service) validateCategory(id string) error {
	if id == "" {
		return nil
	}
	var count int64
	if err := s.store.DB.Model(&model.Category{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: category does not exist", ErrInvalid)
	}
	return nil
}

func slugify(value string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
}

func uniqueTags(values []string) []string {
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && len([]rune(value)) <= 30 && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
		if len(result) == 10 {
			break
		}
	}
	return result
}

func decodePostTags(posts []model.Post) {
	for index := range posts {
		_ = json.Unmarshal([]byte(posts[index].TagsJSON), &posts[index].Tags)
	}
}

func identityName(actor Identity) string {
	if value := strings.TrimSpace(actor.DisplayName); value != "" {
		return value
	}
	if value := strings.TrimSpace(actor.Username); value != "" {
		return value
	}
	return actor.ID
}

func NewID(prefix string) string { return newID(prefix) }

func newID(prefix string) string {
	value := make([]byte, 12)
	_, _ = rand.Read(value)
	if prefix == "" {
		return hex.EncodeToString(value)
	}
	return prefix + "_" + hex.EncodeToString(value)
}
