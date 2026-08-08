export interface User { id: string; username: string; displayName: string }
export type PostStatus = 'draft' | 'published' | 'archived'
export interface Category { id: string; name: string; slug: string; description: string; postCount: number; createdAt: string; updatedAt: string }
export interface Post { id: string; title: string; slug: string; excerpt: string; content: string; coverImageUrl: string; status: PostStatus; categoryId: string; category?: Category; tags: string[]; authorId: string; authorName: string; publishedAt?: string; createdAt: string; updatedAt: string; commentCount: number }
export interface Comment { id: string; postId: string; content: string; authorId: string; authorName: string; createdAt: string }
export interface Media { id: string; filename: string; contentType: string; size: number; url: string }
export interface Dashboard { posts: number; published: number; drafts: number; comments: number; categories: number; recent: Post[] }
export interface PostPage { items: Post[]; total: number; page: number; pageSize: number }
export interface PostInput { title: string; slug: string; excerpt: string; content: string; coverImageUrl: string; status: PostStatus; categoryId: string; tags: string[] }
