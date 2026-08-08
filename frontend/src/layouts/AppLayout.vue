<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChatDotRound, CollectionTag, Document, EditPen, Grid, Medal, Search, Select, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
const route = useRoute(); const router = useRouter(); const auth = useAuthStore()
const title = computed(() => String(route.meta.title || '内部博客'))
const nav = computed(() => [
  { path: '/dashboard', label: '工作台', icon: Grid, visible: true }, { path: '/explore', label: '发现', icon: Search, visible: true },
  { path: '/posts', label: '我的文章', icon: Document, visible: true }, { path: '/leaderboard', label: '积分榜', icon: Medal, visible: true },
  { path: '/reviews', label: '内容审核', icon: Select, visible: auth.canReview }, { path: '/categories', label: '分类', icon: CollectionTag, visible: auth.isAdmin },
  { path: '/comments', label: '评论', icon: ChatDotRound, visible: true },
].filter((item) => item.visible))
async function logout() { await auth.logout(); await router.replace('/login') }
</script>
<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark"><el-icon><EditPen /></el-icon></span><span><strong>内部博客</strong><small>Knowledge Studio</small></span></div>
      <el-menu :default-active="route.path" router><el-menu-item v-for="item in nav" :key="item.path" :index="item.path"><el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span></el-menu-item></el-menu>
      <div class="sidebar-user"><span class="avatar">{{ (auth.user?.displayName || auth.user?.username || 'U').slice(0, 1) }}</span><span><strong>{{ auth.user?.displayName || auth.user?.username }}</strong><small>{{ auth.user?.username }}</small></span><el-tooltip content="退出登录"><el-button text :icon="SwitchButton" aria-label="退出登录" @click="logout" /></el-tooltip></div>
    </aside>
    <section class="workspace"><header class="topbar"><div><p>CONTENT MANAGEMENT</p><h1>{{ title }}</h1></div><el-button v-if="route.name !== 'post-new' && route.name !== 'post-edit'" type="primary" :icon="EditPen" @click="router.push('/posts/new')">写文章</el-button></header><main><RouterView /></main></section>
  </div>
</template>
