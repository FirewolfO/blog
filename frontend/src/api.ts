import axios from 'axios'
import type { Category, Comment, Dashboard, LeaderboardEntry, Media, Post, PostInput, PostPage, User } from '@/types'

const baseURL = import.meta.env.VITE_BLOG_API_BASE_URL || '/api/v1'
const tokenKey = 'blog_access_token'
interface Envelope<T> { code: string; message: string; data: T; requestId: string }

export const client = axios.create({ baseURL, timeout: 20_000 })
client.interceptors.request.use((config) => { const token = getToken(); if (token) config.headers.Authorization = `Bearer ${token}`; return config })
client.interceptors.response.use((response) => response, (error) => { if (error.response?.status === 401 && !String(error.config?.url).includes('/auth/')) { clearToken(); window.location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`) }; return Promise.reject(error) })

const unwrap = async <T>(promise: Promise<{ data: Envelope<T> }>) => (await promise).data.data
export const getToken = () => sessionStorage.getItem(tokenKey) || ''
export const saveToken = (token: string) => sessionStorage.setItem(tokenKey, token)
export const clearToken = () => sessionStorage.removeItem(tokenKey)
export const apiMessage = (error: unknown, fallback = '请求失败') => axios.isAxiosError<Envelope<unknown>>(error) ? error.response?.data?.message || fallback : error instanceof Error ? error.message : fallback

export const blogApi = {
  oauthURL: (redirectUri: string) => unwrap<{ authorizationUrl: string }>(client.get('/auth/oauth/url', { params: { redirectUri } })),
  oauthCallback: (code: string, state: string, redirectURI: string) => unwrap<{ accessToken: string; expiresAt: string; user: User }>(client.post('/auth/oauth/callback', { code, state, redirectURI })),
  me: () => unwrap<{ user: User }>(client.get('/auth/me')),
  logout: () => unwrap<{ loggedOut: boolean }>(client.post('/auth/logout')),
  dashboard: () => unwrap<Dashboard>(client.get('/dashboard')),
  posts: (params: Record<string, string | number>) => unwrap<PostPage>(client.get('/posts', { params })),
  post: (id: string) => unwrap<Post>(client.get(`/posts/${id}`)),
  createPost: (input: PostInput) => unwrap<Post>(client.post('/posts', input)),
  updatePost: (id: string, input: PostInput) => unwrap<Post>(client.put(`/posts/${id}`, input)),
  publishPost: (id: string) => unwrap<Post>(client.post(`/posts/${id}/publish`)),
  deletePost: (id: string) => unwrap<{ deleted: boolean }>(client.delete(`/posts/${id}`)),
  ratePost: (id: string, stars: number) => unwrap<{ stars: number; ratingCount: number; ratingAverage: number }>(client.post(`/posts/${id}/rating`, { stars })),
  categories: () => unwrap<Category[]>(client.get('/categories')),
  createCategory: (input: Pick<Category, 'name' | 'slug' | 'description'>) => unwrap<Category>(client.post('/categories', input)),
  updateCategory: (id: string, input: Pick<Category, 'name' | 'slug' | 'description'>) => unwrap<Category>(client.put(`/categories/${id}`, input)),
  deleteCategory: (id: string) => unwrap<{ deleted: boolean }>(client.delete(`/categories/${id}`)),
  comments: (postId: string) => unwrap<Comment[]>(client.get(`/posts/${postId}/comments`)),
  createComment: (postId: string, content: string) => unwrap<Comment>(client.post(`/posts/${postId}/comments`, { content })),
  deleteComment: (id: string) => unwrap<{ deleted: boolean }>(client.delete(`/comments/${id}`)),
  upload: (file: File) => { const data = new FormData(); data.append('file', file); return unwrap<Media>(client.post('/media', data, { headers: { 'Content-Type': 'multipart/form-data' } })) },
  reviews: () => unwrap<Post[]>(client.get('/reviews')),
  approve: (id: string) => unwrap<Post>(client.post(`/reviews/${id}/approve`)),
  reject: (id: string, note: string) => unwrap<Post>(client.post(`/reviews/${id}/reject`, { note })),
  leaderboard: () => unwrap<LeaderboardEntry[]>(client.get('/leaderboard')),
  recommendations: () => unwrap<Post[]>(client.get('/recommendations')),
}
