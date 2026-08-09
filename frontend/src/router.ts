import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({ history: createWebHistory(), routes: [
  { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { public: true, title: '登录' } },
  { path: '/oauth/callback', name: 'oauth-callback', component: () => import('@/views/OAuthCallbackView.vue'), meta: { public: true, title: '登录中' } },
  { path: '/', component: () => import('@/layouts/AppLayout.vue'), children: [
    { path: '', redirect: '/dashboard' },
    { path: 'dashboard', name: 'dashboard', component: () => import('@/views/DashboardView.vue'), meta: { title: '工作台' } },
	{ path: 'explore', name: 'explore', component: () => import('@/views/ExploreView.vue'), meta: { title: '发现博客' } },
	{ path: 'leaderboard', name: 'leaderboard', component: () => import('@/views/LeaderboardView.vue'), meta: { title: '积分排行榜' } },
	{ path: 'reviews', name: 'reviews', component: () => import('@/views/ReviewsView.vue'), meta: { title: '审核记录' } },
    { path: 'posts', name: 'posts', component: () => import('@/views/PostsView.vue'), meta: { title: '文章管理' } },
    { path: 'posts/new', name: 'post-new', component: () => import('@/views/PostEditorView.vue'), meta: { title: '新建文章' } },
    { path: 'posts/:id/edit', name: 'post-edit', component: () => import('@/views/PostEditorView.vue'), meta: { title: '编辑文章' } },
    { path: 'categories', name: 'categories', component: () => import('@/views/CategoriesView.vue'), meta: { title: '分类管理' } },
    { path: 'comments', name: 'comments', component: () => import('@/views/CommentsView.vue'), meta: { title: '评论管理' } },
  ] },
  { path: '/:pathMatch(.*)*', redirect: '/' },
] })

router.beforeEach(async (to) => { document.title = `${String(to.meta.title || '内部博客')} - 内部博客`; const auth = useAuthStore(); await auth.hydrate(); if (to.meta.public) return auth.authenticated && to.name === 'login' ? '/dashboard' : true; if (!auth.authenticated) return { name: 'login', query: { redirect: to.fullPath } }; if (to.name === 'categories' && !auth.isAdmin) return '/explore'; return true })
export default router
