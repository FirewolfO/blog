package blog

import (
	"errors"
	"testing"

	"blog/internal/model"
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

func user(id, name string) Identity {
	return Identity{ID: id, Username: name, DisplayName: name, Source: "people"}
}
func admin() Identity {
	return Identity{ID: "admin", Username: "admin", Source: "permission", Permissions: []string{PermissionView, PermissionManage}}
}
func reviewer() Identity {
	return Identity{ID: "reviewer", Username: "reviewer", Source: "permission", Permissions: []string{PermissionView, PermissionReview}}
}

func TestModeratedPostKeepsApprovedVersionVisible(t *testing.T) {
	svc := newService(t)
	author := user("author", "作者")
	reader := user("reader", "读者")

	post, err := svc.CreatePost(author, PostInput{Title: "第一版", Content: "old content", Status: model.PostPublished, Tags: []string{"Go"}})
	if err != nil || post.ReviewStatus != model.ReviewPending {
		t.Fatalf("CreatePost() = %#v, %v", post, err)
	}
	if _, err := svc.GetPost(reader, post.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pending post visible: %v", err)
	}
	if _, err := svc.ApprovePost(post.ID, reviewer()); err != nil {
		t.Fatal(err)
	}

	updated, err := svc.UpdatePost(post.ID, author, PostInput{Title: "第二版", Content: "new content", Status: model.PostPublished, Tags: []string{"Go", "SQLite"}})
	if err != nil || !updated.HasPendingChanges || updated.Content != "new content" {
		t.Fatalf("UpdatePost() = %#v, %v", updated, err)
	}
	public, err := svc.GetPost(reader, post.ID)
	if err != nil || public.Title != "第一版" || public.Content != "old content" {
		t.Fatalf("public before review = %#v, %v", public, err)
	}
	if _, err := svc.ApprovePost(post.ID, reviewer()); err != nil {
		t.Fatal(err)
	}
	public, err = svc.GetPost(reader, post.ID)
	if err != nil || public.Title != "第二版" || public.Content != "new content" {
		t.Fatalf("public after review = %#v, %v", public, err)
	}
}

func TestOwnershipCommentAndReviewerRules(t *testing.T) {
	svc := newService(t)
	author, stranger := user("author", "作者"), user("stranger", "路人")
	post, err := svc.CreatePost(admin(), PostInput{Title: "公开文章", Content: "content", Status: model.PostPublished})
	if err != nil {
		t.Fatal(err)
	}
	private, err := svc.CreatePost(author, PostInput{Title: "作者草稿", Content: "draft", Status: model.PostDraft})
	if err != nil {
		t.Fatal(err)
	}
	adminPage, err := svc.ListPosts(admin(), "", "", "", "", 1, 20)
	if err != nil || adminPage.Total != 2 {
		t.Fatalf("administrator list = %#v, %v", adminPage, err)
	}
	if _, err := svc.GetPost(stranger, private.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("private draft visible to stranger: %v", err)
	}
	if _, err := svc.UpdatePost(post.ID, stranger, PostInput{Title: "越权", Content: "bad"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdatePost error = %v", err)
	}
	comment, err := svc.CreateComment(post.ID, stranger, "评论")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteComment(comment.ID, author); !errors.Is(err, ErrForbidden) {
		t.Fatalf("DeleteComment error = %v", err)
	}
	if err := svc.DeleteComment(comment.ID, stranger); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ApprovePost(post.ID, author); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ApprovePost error = %v", err)
	}
}

func TestSearchRatingsLeaderboardAndRecommendations(t *testing.T) {
	svc := newService(t)
	alice, bob, carol := user("alice", "Alice"), user("bob", "Bob"), user("carol", "Carol")
	first, _ := svc.CreatePost(admin(), PostInput{Title: "Gateway 实践", Content: "Go routing", Status: model.PostPublished, Tags: []string{"Go", "Gateway"}})
	// Assign an author that can receive points.
	svc.store.DB.Model(&model.Post{}).Where("id = ?", first.ID).Updates(map[string]any{"author_id": alice.ID, "author_name": alice.DisplayName})
	second, _ := svc.CreatePost(admin(), PostInput{Title: "SQLite 调优", Content: "database indexes", Status: model.PostPublished, Tags: []string{"SQLite"}})
	svc.store.DB.Model(&model.Post{}).Where("id = ?", second.ID).Updates(map[string]any{"author_id": bob.ID, "author_name": bob.DisplayName})

	page, err := svc.ListPosts(carol, "Alice", "published", "", "public", 1, 20)
	if err != nil || page.Total != 1 || page.Items[0].ID != first.ID {
		t.Fatalf("author search = %#v, %v", page, err)
	}
	if _, err := svc.RatePost(first.ID, bob, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RatePost(first.ID, carol, 4); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateComment(first.ID, carol, "很好"); err != nil {
		t.Fatal(err)
	}
	leaders, err := svc.Leaderboard(10)
	if err != nil || len(leaders) == 0 || leaders[0].AuthorID != alice.ID || leaders[0].Score <= 0 {
		t.Fatalf("leaderboard = %#v, %v", leaders, err)
	}
	recommendations, err := svc.Recommendations(carol, 5)
	if err != nil || len(recommendations) == 0 || recommendations[0].RecommendationReason == "" {
		t.Fatalf("recommendations = %#v, %v", recommendations, err)
	}
}
