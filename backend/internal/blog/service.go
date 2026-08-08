package blog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"blog/internal/model"
	"blog/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PermissionView   = "svc.inner.blog:view"
	PermissionManage = "svc.inner.blog:manage"
	PermissionReview = "blog.review:manage"
)

var (
	ErrInvalid   = errors.New("invalid input")
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
	nonSlug      = regexp.MustCompile(`[^a-z0-9]+`)
)

type Identity struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Source      string   `json:"source,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func (i Identity) Has(permission string) bool {
	for _, value := range i.Permissions {
		if value == "*" || value == permission {
			return true
		}
	}
	return false
}

// People OAuth users are regular Blog users. Permission identities also need
// the baseline view permission before the BFF forwards a request.
func (i Identity) Can(permission string) bool {
	return i.Source != "permission" || i.Has(permission)
}
func (i Identity) IsAdmin() bool   { return i.Has(PermissionManage) }
func (i Identity) CanReview() bool { return i.IsAdmin() || i.Has(PermissionReview) }

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
	Pending    int64        `json:"pending"`
	Comments   int64        `json:"comments"`
	Categories int64        `json:"categories"`
	Recent     []model.Post `json:"recent"`
}
type LeaderboardEntry struct {
	Rank           int     `json:"rank"`
	AuthorID       string  `json:"authorId"`
	AuthorName     string  `json:"authorName"`
	Score          int     `json:"score"`
	PublishedPosts int64   `json:"publishedPosts"`
	RatingCount    int64   `json:"ratingCount"`
	TotalStars     int64   `json:"totalStars"`
	AverageRating  float64 `json:"averageRating"`
	CommentCount   int64   `json:"commentCount"`
}
type RatingResult struct {
	Stars         int     `json:"stars"`
	RatingCount   int64   `json:"ratingCount"`
	RatingAverage float64 `json:"ratingAverage"`
}

type Service struct{ store *store.Store }

func New(database *store.Store) *Service { return &Service{store: database} }

func (s *Service) Dashboard(actor Identity) (*Dashboard, error) {
	result := &Dashboard{Recent: []model.Post{}}
	posts := s.visiblePostsQuery(actor, "")
	if err := posts.Count(&result.Posts).Error; err != nil {
		return nil, err
	}
	if err := s.visiblePostsQuery(actor, "").Where("status = ? AND review_status = ?", model.PostPublished, model.ReviewApproved).Count(&result.Published).Error; err != nil {
		return nil, err
	}
	if err := s.visiblePostsQuery(actor, "").Where("status = ?", model.PostDraft).Count(&result.Drafts).Error; err != nil {
		return nil, err
	}
	if actor.CanReview() {
		if err := s.store.DB.Model(&model.Post{}).Where("review_status = ? OR pending_revision_id <> ''", model.ReviewPending).Count(&result.Pending).Error; err != nil {
			return nil, err
		}
	}
	if actor.IsAdmin() {
		_ = s.store.DB.Model(&model.Comment{}).Count(&result.Comments).Error
	} else {
		_ = s.store.DB.Model(&model.Comment{}).Where("post_id IN (?)", s.visiblePostsQuery(actor, "").Select("id")).Count(&result.Comments).Error
	}
	_ = s.store.DB.Model(&model.Category{}).Count(&result.Categories).Error
	if err := s.visiblePostsQuery(actor, "").Preload("Category").Order("updated_at DESC").Limit(5).Find(&result.Recent).Error; err != nil {
		return nil, err
	}
	for index := range result.Recent {
		s.decoratePost(&result.Recent[index], actor, result.Recent[index].AuthorID == actor.ID)
	}
	return result, nil
}

func (s *Service) ListPosts(actor Identity, search, status, categoryID, scope string, page, pageSize int) (*PostPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.visiblePostsQuery(actor, scope).Preload("Category")
	if value := strings.TrimSpace(search); value != "" {
		like := "%" + value + "%"
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR content LIKE ? OR author_name LIKE ?", like, like, like, like)
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
		s.decoratePost(&result.Items[index], actor, scope != "public" && result.Items[index].AuthorID == actor.ID)
	}
	return result, nil
}

func (s *Service) visiblePostsQuery(actor Identity, scope string) *gorm.DB {
	query := s.store.DB.Model(&model.Post{})
	switch scope {
	case "public":
		return query.Where("status = ? AND review_status = ?", model.PostPublished, model.ReviewApproved)
	case "mine":
		return query.Where("author_id = ?", actor.ID)
	case "all":
		if actor.IsAdmin() {
			return query
		}
	}
	if actor.IsAdmin() {
		return query
	}
	return query.Where("author_id = ? OR (status = ? AND review_status = ?)", actor.ID, model.PostPublished, model.ReviewApproved)
}

func (s *Service) GetPost(actor Identity, id string) (*model.Post, error) {
	var post model.Post
	if err := s.store.DB.Preload("Category").Where("id = ?", id).First(&post).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	owner := post.AuthorID == actor.ID
	if !owner && !actor.IsAdmin() && !actor.CanReview() && !(post.Status == model.PostPublished && post.ReviewStatus == model.ReviewApproved) {
		return nil, ErrNotFound
	}
	s.decoratePost(&post, actor, owner || actor.IsAdmin() || actor.CanReview())
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
	post.ReviewStatus = model.ReviewDraft
	if post.Status == model.PostPublished {
		if actor.IsAdmin() {
			post.ReviewStatus = model.ReviewApproved
			now := time.Now().UTC()
			post.PublishedAt = &now
		} else {
			post.ReviewStatus = model.ReviewPending
			post.PublishedAt = nil
		}
	}
	if err := s.store.DB.Create(post).Error; err != nil {
		return nil, err
	}
	return s.GetPost(actor, post.ID)
}

func (s *Service) UpdatePost(id string, actor Identity, input PostInput) (*model.Post, error) {
	current, err := s.getStoredPost(id)
	if err != nil {
		return nil, err
	}
	if current.AuthorID != actor.ID && !actor.IsAdmin() {
		return nil, ErrForbidden
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
	if !actor.IsAdmin() && current.Status == model.PostPublished && current.ReviewStatus == model.ReviewApproved && input.Status != model.PostArchived {
		revision := revisionFromPost(current.ID, actor, updated)
		if current.PendingRevisionID != "" {
			revision.ID = current.PendingRevisionID
			if err := s.store.DB.Where("id = ?", revision.ID).Assign(revision).FirstOrCreate(revision).Error; err != nil {
				return nil, err
			}
		} else {
			revision.ID = newID("revision")
			if err := s.store.DB.Create(revision).Error; err != nil {
				return nil, err
			}
		}
		if err := s.store.DB.Model(&model.Post{}).Where("id = ?", id).Updates(map[string]any{"pending_revision_id": revision.ID, "review_note": ""}).Error; err != nil {
			return nil, err
		}
		return s.GetPost(actor, id)
	}
	reviewStatus := model.ReviewDraft
	var publishedAt *time.Time
	if updated.Status == model.PostPublished {
		if actor.IsAdmin() {
			reviewStatus = model.ReviewApproved
			now := time.Now().UTC()
			publishedAt = &now
		} else {
			reviewStatus = model.ReviewPending
		}
	}
	values := postValues(updated)
	values["review_status"] = reviewStatus
	values["review_note"] = ""
	values["published_at"] = publishedAt
	values["pending_revision_id"] = ""
	if err := s.store.DB.Model(&model.Post{}).Where("id = ?", id).Updates(values).Error; err != nil {
		return nil, err
	}
	return s.GetPost(actor, id)
}

func (s *Service) PublishPost(id string, actor Identity) (*model.Post, error) {
	post, err := s.getStoredPost(id)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	if post.PendingRevisionID != "" {
		return s.GetPost(actor, id)
	}
	values := map[string]any{"status": model.PostPublished, "review_note": ""}
	if actor.IsAdmin() {
		now := time.Now().UTC()
		values["review_status"] = model.ReviewApproved
		values["published_at"] = &now
	} else {
		values["review_status"] = model.ReviewPending
		values["published_at"] = nil
	}
	if err := s.store.DB.Model(&model.Post{}).Where("id = ?", id).Updates(values).Error; err != nil {
		return nil, err
	}
	return s.GetPost(actor, id)
}

func (s *Service) ListReviews(actor Identity) ([]model.Post, error) {
	if !actor.CanReview() {
		return nil, ErrForbidden
	}
	items := []model.Post{}
	if err := s.store.DB.Preload("Category").Where("review_status = ? OR pending_revision_id <> ''", model.ReviewPending).Order("updated_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	result := items[:0]
	for index := range items {
		s.decoratePost(&items[index], actor, true)
		if items[index].ReviewStatus == model.ReviewPending {
			result = append(result, items[index])
		}
	}
	return result, nil
}

func (s *Service) ApprovePost(id string, actor Identity) (*model.Post, error) {
	if !actor.CanReview() {
		return nil, ErrForbidden
	}
	post, err := s.getStoredPost(id)
	if err != nil {
		return nil, err
	}
	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if post.PendingRevisionID != "" {
			var revision model.PostRevision
			if err := tx.Where("id = ? AND review_status = ?", post.PendingRevisionID, model.ReviewPending).First(&revision).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrConflict
			} else if err != nil {
				return err
			}
			values := revisionValues(&revision)
			values["review_status"] = model.ReviewApproved
			values["review_note"] = ""
			values["pending_revision_id"] = ""
			if revision.Status == model.PostPublished {
				values["published_at"] = &now
			} else {
				values["published_at"] = nil
			}
			if err := tx.Model(&model.Post{}).Where("id = ?", id).Updates(values).Error; err != nil {
				return err
			}
			return tx.Model(&revision).Updates(map[string]any{"review_status": model.ReviewApproved, "reviewed_by": actor.ID, "reviewed_at": &now, "review_note": ""}).Error
		}
		if post.ReviewStatus != model.ReviewPending {
			return ErrConflict
		}
		values := map[string]any{"review_status": model.ReviewApproved, "review_note": ""}
		if post.Status == model.PostPublished {
			values["published_at"] = &now
		}
		return tx.Model(post).Updates(values).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetPost(actor, id)
}

func (s *Service) RejectPost(id string, actor Identity, note string) (*model.Post, error) {
	if !actor.CanReview() {
		return nil, ErrForbidden
	}
	note = strings.TrimSpace(note)
	if note == "" || len([]rune(note)) > 500 {
		return nil, fmt.Errorf("%w: review note is required", ErrInvalid)
	}
	post, err := s.getStoredPost(id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if post.PendingRevisionID != "" {
		var revision model.PostRevision
		if err := s.store.DB.Where("id = ? AND review_status = ?", post.PendingRevisionID, model.ReviewPending).First(&revision).Error; err != nil {
			return nil, ErrConflict
		}
		if err := s.store.DB.Model(&revision).Updates(map[string]any{"review_status": model.ReviewRejected, "reviewed_by": actor.ID, "reviewed_at": &now, "review_note": note}).Error; err != nil {
			return nil, err
		}
		_ = s.store.DB.Model(post).Update("review_note", note).Error
		return s.GetPost(actor, id)
	}
	if post.ReviewStatus != model.ReviewPending {
		return nil, ErrConflict
	}
	if err := s.store.DB.Model(post).Updates(map[string]any{"review_status": model.ReviewRejected, "review_note": note}).Error; err != nil {
		return nil, err
	}
	return s.GetPost(actor, id)
}

func (s *Service) DeletePost(id string, actor Identity) error {
	post, err := s.getStoredPost(id)
	if err != nil {
		return err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() {
		return ErrForbidden
	}
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		for _, target := range []any{&model.Comment{}, &model.Rating{}, &model.PostRevision{}} {
			if err := tx.Where("post_id = ?", id).Delete(target).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ?", id).Delete(&model.Post{}).Error
	})
}

func (s *Service) ListCategories() ([]model.Category, error) {
	result := []model.Category{}
	if err := s.store.DB.Order("name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	for i := range result {
		_ = s.store.DB.Model(&model.Post{}).Where("category_id = ? AND status = ? AND review_status = ?", result[i].ID, model.PostPublished, model.ReviewApproved).Count(&result[i].PostCount).Error
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

func (s *Service) ListComments(postID string, actor Identity) ([]model.Comment, error) {
	post, err := s.GetPost(actor, postID)
	if err != nil {
		return nil, err
	}
	result := []model.Comment{}
	if err := s.store.DB.Where("post_id = ?", postID).Order("created_at DESC").Find(&result).Error; err != nil {
		return nil, err
	}
	for index := range result {
		result[index].CanDelete = result[index].AuthorID == actor.ID || post.AuthorID == actor.ID || actor.IsAdmin()
	}
	return result, nil
}
func (s *Service) CreateComment(postID string, actor Identity, content string) (*model.Comment, error) {
	post, err := s.GetPost(actor, postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() && !(post.Status == model.PostPublished && post.ReviewStatus == model.ReviewApproved) {
		return nil, ErrForbidden
	}
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 1000 {
		return nil, fmt.Errorf("%w: comment content is required and limited to 1000 characters", ErrInvalid)
	}
	comment := &model.Comment{ID: newID("comment"), PostID: postID, Content: content, AuthorID: actor.ID, AuthorName: identityName(actor)}
	return comment, s.store.DB.Create(comment).Error
}
func (s *Service) DeleteComment(id string, actor Identity) error {
	var comment model.Comment
	if err := s.store.DB.Where("id = ?", id).First(&comment).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	post, err := s.getStoredPost(comment.PostID)
	if err != nil {
		return err
	}
	if comment.AuthorID != actor.ID && post.AuthorID != actor.ID && !actor.IsAdmin() {
		return ErrForbidden
	}
	return s.store.DB.Delete(&comment).Error
}

func (s *Service) RatePost(postID string, actor Identity, stars int) (*RatingResult, error) {
	if stars < 1 || stars > 5 {
		return nil, fmt.Errorf("%w: stars must be between 1 and 5", ErrInvalid)
	}
	post, err := s.getStoredPost(postID)
	if err != nil {
		return nil, err
	}
	if post.Status != model.PostPublished || post.ReviewStatus != model.ReviewApproved {
		return nil, ErrNotFound
	}
	if post.AuthorID == actor.ID {
		return nil, fmt.Errorf("%w: authors cannot rate their own posts", ErrInvalid)
	}
	rating := model.Rating{ID: newID("rating"), PostID: postID, RaterID: actor.ID, Stars: stars}
	if err := s.store.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "post_id"}, {Name: "rater_id"}}, DoUpdates: clause.AssignmentColumns([]string{"stars", "updated_at"})}).Create(&rating).Error; err != nil {
		return nil, err
	}
	count, avg := s.ratingSummary(postID)
	return &RatingResult{Stars: stars, RatingCount: count, RatingAverage: avg}, nil
}

func (s *Service) Leaderboard(limit int) ([]LeaderboardEntry, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	posts := []model.Post{}
	if err := s.store.DB.Where("status = ? AND review_status = ?", model.PostPublished, model.ReviewApproved).Find(&posts).Error; err != nil {
		return nil, err
	}
	byAuthor := map[string]*LeaderboardEntry{}
	for _, post := range posts {
		entry := byAuthor[post.AuthorID]
		if entry == nil {
			entry = &LeaderboardEntry{AuthorID: post.AuthorID, AuthorName: post.AuthorName}
			byAuthor[post.AuthorID] = entry
		}
		entry.PublishedPosts++
		var ratings struct {
			Count   int64
			Total   int64
			Average float64
		}
		_ = s.store.DB.Model(&model.Rating{}).Select("COUNT(*) count, COALESCE(SUM(stars),0) total, COALESCE(AVG(stars),0) average").Where("post_id = ?", post.ID).Scan(&ratings).Error
		entry.RatingCount += ratings.Count
		entry.TotalStars += ratings.Total
		var comments int64
		_ = s.store.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&comments).Error
		entry.CommentCount += comments
	}
	result := make([]LeaderboardEntry, 0, len(byAuthor))
	for _, entry := range byAuthor {
		if entry.RatingCount > 0 {
			entry.AverageRating = math.Round(float64(entry.TotalStars)/float64(entry.RatingCount)*10) / 10
		}
		entry.Score = int(entry.PublishedPosts*20+entry.RatingCount*3+entry.TotalStars*2+entry.CommentCount) + int(math.Round(entry.AverageRating*10))
		result = append(result, *entry)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].AuthorName < result[j].AuthorName
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > limit {
		result = result[:limit]
	}
	for i := range result {
		result[i].Rank = i + 1
	}
	return result, nil
}

func (s *Service) Recommendations(actor Identity, limit int) ([]model.Post, error) {
	if limit < 1 || limit > 20 {
		limit = 6
	}
	own := []model.Post{}
	_ = s.store.DB.Where("author_id = ?", actor.ID).Find(&own).Error
	interests := map[string]bool{}
	for _, post := range own {
		var tags []string
		_ = json.Unmarshal([]byte(post.TagsJSON), &tags)
		for _, tag := range tags {
			interests[strings.ToLower(tag)] = true
		}
	}
	candidates := []model.Post{}
	if err := s.store.DB.Preload("Category").Where("status = ? AND review_status = ? AND author_id <> ?", model.PostPublished, model.ReviewApproved, actor.ID).Limit(100).Find(&candidates).Error; err != nil {
		return nil, err
	}
	type ranked struct {
		post  model.Post
		score float64
	}
	rankedPosts := make([]ranked, 0, len(candidates))
	for _, post := range candidates {
		s.decoratePost(&post, actor, false)
		score := post.RatingAverage*10 + float64(post.RatingCount*2+post.CommentCount)
		matches := 0
		for _, tag := range post.Tags {
			if interests[strings.ToLower(tag)] {
				matches++
			}
		}
		score += float64(matches * 8)
		days := time.Since(post.UpdatedAt).Hours() / 24
		if days < 30 {
			score += float64(30-days) / 3
		}
		if matches > 0 {
			post.RecommendationReason = fmt.Sprintf("与你关注的 %d 个标签相关", matches)
		} else if post.RatingCount > 0 {
			post.RecommendationReason = fmt.Sprintf("读者评分 %.1f", post.RatingAverage)
		} else {
			post.RecommendationReason = "近期发布"
		}
		rankedPosts = append(rankedPosts, ranked{post, score})
	}
	sort.SliceStable(rankedPosts, func(i, j int) bool { return rankedPosts[i].score > rankedPosts[j].score })
	if len(rankedPosts) > limit {
		rankedPosts = rankedPosts[:limit]
	}
	result := make([]model.Post, len(rankedPosts))
	for i := range rankedPosts {
		result[i] = rankedPosts[i].post
	}
	return result, nil
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

func (s *Service) getStoredPost(id string) (*model.Post, error) {
	var post model.Post
	if err := s.store.DB.Preload("Category").Where("id = ?", id).First(&post).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &post, nil
}
func (s *Service) decoratePost(post *model.Post, actor Identity, includePending bool) {
	_ = json.Unmarshal([]byte(post.TagsJSON), &post.Tags)
	_ = s.store.DB.Model(&model.Comment{}).Where("post_id = ?", post.ID).Count(&post.CommentCount).Error
	post.RatingCount, post.RatingAverage = s.ratingSummary(post.ID)
	_ = s.store.DB.Model(&model.Rating{}).Select("stars").Where("post_id = ? AND rater_id = ?", post.ID, actor.ID).Scan(&post.MyRating).Error
	post.CanEdit = post.AuthorID == actor.ID || actor.IsAdmin()
	post.CanDelete = post.CanEdit
	if includePending && post.PendingRevisionID != "" {
		var revision model.PostRevision
		if s.store.DB.Where("id = ?", post.PendingRevisionID).First(&revision).Error == nil {
			overlayRevision(post, &revision)
		}
	}
}
func (s *Service) ratingSummary(postID string) (int64, float64) {
	var result struct {
		Count   int64
		Average float64
	}
	_ = s.store.DB.Model(&model.Rating{}).Select("COUNT(*) count, COALESCE(AVG(stars),0) average").Where("post_id = ?", postID).Scan(&result).Error
	return result.Count, math.Round(result.Average*10) / 10
}

func overlayRevision(post *model.Post, revision *model.PostRevision) {
	post.Title = revision.Title
	post.Slug = revision.Slug
	post.Excerpt = revision.Excerpt
	post.Content = revision.Content
	post.CoverImageURL = revision.CoverImageURL
	post.Status = revision.Status
	post.CategoryID = revision.CategoryID
	post.TagsJSON = revision.TagsJSON
	_ = json.Unmarshal([]byte(revision.TagsJSON), &post.Tags)
	post.ReviewStatus = revision.ReviewStatus
	post.ReviewNote = revision.ReviewNote
	post.HasPendingChanges = true
}
func revisionFromPost(postID string, actor Identity, post *model.Post) *model.PostRevision {
	return &model.PostRevision{PostID: postID, Title: post.Title, Slug: post.Slug, Excerpt: post.Excerpt, Content: post.Content, CoverImageURL: post.CoverImageURL, Status: post.Status, CategoryID: post.CategoryID, TagsJSON: post.TagsJSON, SubmittedBy: actor.ID, SubmittedName: identityName(actor), ReviewStatus: model.ReviewPending}
}
func postValues(post *model.Post) map[string]any {
	return map[string]any{"title": post.Title, "slug": post.Slug, "excerpt": post.Excerpt, "content": post.Content, "cover_image_url": post.CoverImageURL, "status": post.Status, "category_id": post.CategoryID, "tags": post.TagsJSON}
}
func revisionValues(revision *model.PostRevision) map[string]any {
	return map[string]any{"title": revision.Title, "slug": revision.Slug, "excerpt": revision.Excerpt, "content": revision.Content, "cover_image_url": revision.CoverImageURL, "status": revision.Status, "category_id": revision.CategoryID, "tags": revision.TagsJSON}
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
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
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
