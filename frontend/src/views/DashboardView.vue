<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ChatDotRound, CollectionTag, DocumentChecked, EditPen, Files } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { apiMessage, blogApi } from '@/api'
import type { Dashboard } from '@/types'
const data = ref<Dashboard | null>(null); const loading = ref(true)
const metrics = () => [
  { label: '全部文章', value: data.value?.posts || 0, icon: Files, tone: 'green' }, { label: '已发布', value: data.value?.published || 0, icon: DocumentChecked, tone: 'blue' },
  { label: '草稿', value: data.value?.drafts || 0, icon: EditPen, tone: 'amber' }, { label: '评论', value: data.value?.comments || 0, icon: ChatDotRound, tone: 'coral' }, { label: '分类', value: data.value?.categories || 0, icon: CollectionTag, tone: 'violet' },
]
onMounted(async () => { try { data.value = await blogApi.dashboard() } catch (error) { ElMessage.error(apiMessage(error, '工作台加载失败')) } finally { loading.value = false } })
const date = (value: string) => new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium' }).format(new Date(value))
</script>
<template><div v-loading="loading" class="page-stack"><section class="metric-grid"><article v-for="item in metrics()" :key="item.label" class="metric"><span :class="`metric-${item.tone}`"><el-icon><component :is="item.icon" /></el-icon></span><div><strong>{{ item.value }}</strong><small>{{ item.label }}</small></div></article></section><section class="data-section"><header class="section-heading"><div><h2>最近更新</h2><p>快速回到正在推进的内容。</p></div><el-button @click="$router.push('/posts')">查看全部</el-button></header><el-table :data="data?.recent || []" empty-text="还没有文章"><el-table-column label="文章" min-width="280"><template #default="scope"><div class="post-cell"><strong>{{ scope.row.title }}</strong><small>{{ scope.row.authorName }} · {{ date(scope.row.updatedAt) }}</small></div></template></el-table-column><el-table-column prop="category.name" label="分类" width="150"><template #default="scope">{{ scope.row.category?.name || '未分类' }}</template></el-table-column><el-table-column label="状态" width="110"><template #default="scope"><el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'archived' ? 'info' : 'warning'" effect="plain">{{ scope.row.status === 'published' ? '已发布' : scope.row.status === 'archived' ? '已归档' : '草稿' }}</el-tag></template></el-table-column><el-table-column width="90"><template #default="scope"><el-button link type="primary" @click="$router.push(`/posts/${scope.row.id}/edit`)">编辑</el-button></template></el-table-column></el-table></section></div></template>
