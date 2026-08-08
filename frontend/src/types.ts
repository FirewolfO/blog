export interface User { id: string; username: string; displayName: string; permissions?: string[] }
export type PostStatus = 'draft' | 'published' | 'archived'
export type ReviewStatus = 'draft' | 'pending' | 'approved' | 'rejected'
export interface Category { id: string; name: string; slug: string; description: string; postCount: number; createdAt: string; updatedAt: string }
export interface Post { id: string; title: string; slug: string; excerpt: string; content: string; coverImageUrl: string; status: PostStatus; reviewStatus: ReviewStatus; reviewNote?: string; hasPendingChanges: boolean; categoryId: string; category?: Category; tags: string[]; authorId: string; authorName: string; publishedAt?: string; createdAt: string; updatedAt: string; commentCount: number; ratingCount: number; ratingAverage: number; myRating: number; canEdit: boolean; canDelete: boolean; recommendationReason?: string }
export interface Comment { id: string; postId: string; content: string; authorId: string; authorName: string; createdAt: string; canDelete: boolean }
export interface Media { id: string; filename: string; contentType: string; size: number; url: string }
export interface Dashboard { posts: number; published: number; drafts: number; pending: number; comments: number; categories: number; recent: Post[] }
export interface PostPage { items: Post[]; total: number; page: number; pageSize: number }
export interface PostInput { title: string; slug: string; excerpt: string; content: string; coverImageUrl: string; status: PostStatus; categoryId: string; tags: string[] }
export interface LeaderboardEntry { rank: number; authorId: string; authorName: string; score: number; publishedPosts: number; ratingCount: number; totalStars: number; averageRating: number; commentCount: number }
