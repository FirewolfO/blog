package blog

import (
	"errors"
	"testing"

	"blog/internal/store"
)

func newService(t *testing.T) *Service {
	t.Helper()
	database, err := store.Open("file:" + t.Name() + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return New(database)
}

func TestPostLifecycleAndDashboard(t *testing.T) {
	svc := newService(t)
	actor := Identity{ID: "pep_admin", Username: "admin", DisplayName: "管理员"}
	category, err := svc.CreateCategory(CategoryInput{Name: "工程实践", Slug: "engineering"})
	if err != nil {
		t.Fatal(err)
	}
	post, err := svc.CreatePost(actor, PostInput{Title: "Gateway Inner 实践", Content: "正文", CategoryID: category.ID, Tags: []string{"Go", "Gateway", "Go"}})
	if err != nil {
		t.Fatal(err)
	}
	if post.Status != "draft" || len(post.Tags) != 2 || post.AuthorID != actor.ID {
		t.Fatalf("post = %#v", post)
	}
	comment, err := svc.CreateComment(post.ID, actor, "内容很有帮助")
	if err != nil || comment.PostID != post.ID {
		t.Fatalf("comment = %#v, %v", comment, err)
	}
	published, err := svc.PublishPost(post.ID)
	if err != nil || published.PublishedAt == nil || published.Status != "published" {
		t.Fatalf("published = %#v, %v", published, err)
	}
	dashboard, err := svc.Dashboard()
	if err != nil || dashboard.Posts != 1 || dashboard.Published != 1 || dashboard.Comments != 1 {
		t.Fatalf("dashboard = %#v, %v", dashboard, err)
	}
	page, err := svc.ListPosts("Inner", "published", category.ID, 1, 20)
	if err != nil || page.Total != 1 || page.Items[0].CommentCount != 1 {
		t.Fatalf("page = %#v, %v", page, err)
	}
	if err := svc.DeletePost(post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPost(post.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetPost() error = %v", err)
	}
}

func TestValidationAndCategoryInUse(t *testing.T) {
	svc := newService(t)
	actor := Identity{ID: "user", Username: "user"}
	if _, err := svc.CreatePost(actor, PostInput{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("CreatePost() error = %v", err)
	}
	category, err := svc.CreateCategory(CategoryInput{Name: "平台", Slug: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateCategory(CategoryInput{Name: "平台", Slug: "another"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate category error = %v", err)
	}
	if _, err := svc.CreatePost(actor, PostInput{Title: "标题", Slug: "first", Content: "正文", CategoryID: category.ID}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteCategory(category.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("DeleteCategory() error = %v", err)
	}
}
