<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Bell, ChatDotRound, CollectionTag, Document, EditPen, Grid, Medal, Search, Select, SwitchButton } from '@element-plus/icons-vue'
import { ElNotification } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { blogApi } from '@/api'
const route = useRoute(); const router = useRouter(); const auth = useAuthStore()
const title = computed(() => String(route.meta.title || '内部博客'))
const pendingReviews = ref(0)
let previousReviews: number | null = null
let reviewTimer = 0
const nav = computed(() => [
  { path: '/dashboard', label: '工作台', icon: Grid, visible: true }, { path: '/explore', label: '发现', icon: Search, visible: true },
  { path: '/posts', label: '我的文章', icon: Document, visible: true }, { path: '/leaderboard', label: '积分榜', icon: Medal, visible: true },
  { path: '/reviews', label: '内容审核', icon: Select, visible: auth.canReview }, { path: '/categories', label: '分类', icon: CollectionTag, visible: auth.isAdmin },
  { path: '/comments', label: '评论', icon: ChatDotRound, visible: true },
].filter((item) => item.visible))
async function logout() { await auth.logout(); await router.replace('/login') }

async function loadReviewCount() {
  if (!auth.canReview) return
  try {
    const count = (await blogApi.reviews()).length
    if (previousReviews !== null && count > previousReviews) ElNotification({ title: '新的内容待审核', message: `审核队列新增 ${count - previousReviews} 篇文章`, type: 'warning', duration: 6000 })
    previousReviews = count
    pendingReviews.value = count
  } catch {
    // Polling is advisory and must not interrupt editing or reading.
  }
}

onMounted(() => { void loadReviewCount(); reviewTimer = window.setInterval(loadReviewCount, 10_000) })
onBeforeUnmount(() => window.clearInterval(reviewTimer))
</script>
<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark"><el-icon><EditPen /></el-icon></span><span><strong>内部博客</strong><small>Knowledge Studio</small></span></div>
      <el-menu :default-active="route.path" router><el-menu-item v-for="item in nav" :key="item.path" :index="item.path"><el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span></el-menu-item></el-menu>
      <div class="sidebar-user"><span class="avatar">{{ (auth.user?.displayName || auth.user?.username || 'U').slice(0, 1) }}</span><span><strong>{{ auth.user?.displayName || auth.user?.username }}</strong><small>{{ auth.user?.username }}</small></span><el-tooltip content="退出登录"><el-button text :icon="SwitchButton" aria-label="退出登录" @click="logout" /></el-tooltip></div>
    </aside>
    <section class="workspace"><header class="topbar"><div><p>CONTENT MANAGEMENT</p><h1>{{ title }}</h1></div><div class="topbar-actions"><el-tooltip v-if="auth.canReview" content="待审核内容"><el-badge :value="pendingReviews" :hidden="pendingReviews === 0" :max="99"><el-button :icon="Bell" circle aria-label="待审核内容" @click="router.push('/reviews')" /></el-badge></el-tooltip><el-button v-if="route.name !== 'post-new' && route.name !== 'post-edit'" type="primary" :icon="EditPen" @click="router.push('/posts/new')">写文章</el-button></div></header><main><RouterView /></main></section>
  </div>
</template>
