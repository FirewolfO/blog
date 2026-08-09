<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { ChatDotRound, Search, Star } from '@element-plus/icons-vue'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import { apiMessage, blogApi } from '@/api'
import EmojiPicker from '@/components/EmojiPicker.vue'
import { useAuthStore } from '@/stores/auth'
import type { Comment, Post, PostPage } from '@/types'

const auth = useAuthStore()
const query = ref('')
const loading = ref(false)
const recommendations = ref<Post[]>([])
const page = reactive<PostPage>({ items: [], total: 0, page: 1, pageSize: 12 })
const selected = ref<Post>()
const comments = ref<Comment[]>([])
const comment = ref('')
const commentInput = ref<{ textarea?: HTMLTextAreaElement } | null>(null)
const sending = ref(false)
const ratingSubmitting = ref(false)

const canRate = computed(() => Boolean(selected.value && selected.value.authorId !== auth.user?.id))
const ratingTooltip = computed(() => canRate.value ? '星标评分' : '不能为自己的文章评分')

async function search() {
  loading.value = true
  try {
    Object.assign(page, await blogApi.posts({ q: query.value, scope: 'public', page: page.page, pageSize: page.pageSize }))
  } catch (error) {
    ElMessage.error(apiMessage(error, '博客加载失败'))
  } finally {
    loading.value = false
  }
}

async function open(post: Post) {
  try {
    selected.value = await blogApi.post(post.id)
    comments.value = await blogApi.comments(post.id)
    comment.value = ''
  } catch (error) {
    ElMessage.error(apiMessage(error, '文章加载失败'))
  }
}

async function rate(value: number | undefined) {
  if (!selected.value || !value || !canRate.value) return
  ratingSubmitting.value = true
  try {
    const result = await blogApi.ratePost(selected.value.id, value)
    selected.value.myRating = result.stars
    selected.value.ratingCount = result.ratingCount
    selected.value.ratingAverage = result.ratingAverage
    ElMessage.success('评分已更新')
    await search()
  } catch (error) {
    ElMessage.error(apiMessage(error, '评分失败'))
  } finally {
    ratingSubmitting.value = false
  }
}

async function insertEmoji(emoji: string) {
  const textarea = commentInput.value?.textarea
  const start = textarea?.selectionStart ?? comment.value.length
  const end = textarea?.selectionEnd ?? start
  comment.value = `${comment.value.slice(0, start)}${emoji}${comment.value.slice(end)}`
  await nextTick()
  const cursor = start + emoji.length
  textarea?.focus()
  textarea?.setSelectionRange(cursor, cursor)
}

async function send() {
  if (!selected.value || !comment.value.trim()) return
  sending.value = true
  try {
    await blogApi.createComment(selected.value.id, comment.value)
    comment.value = ''
    comments.value = await blogApi.comments(selected.value.id)
    ElMessage.success('评论已发布')
  } catch (error) {
    ElMessage.error(apiMessage(error, '评论失败'))
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  await search()
  try {
    recommendations.value = await blogApi.recommendations()
  } catch {
    recommendations.value = []
  }
})
</script>

<template>
  <div class="page-stack explore-page">
    <section v-if="recommendations.length" class="data-section">
      <div class="section-heading"><div><h2>为你推荐</h2><p>结合评分、热度和你关注的标签</p></div></div>
      <div class="recommendation-strip">
        <button v-for="post in recommendations" :key="post.id" @click="open(post)">
          <span>{{ post.recommendationReason }}</span><strong>{{ post.title }}</strong><small>{{ post.authorName }} · {{ post.ratingAverage.toFixed(1) }} 分</small>
        </button>
      </div>
    </section>
    <section class="filter-bar">
      <el-input v-model="query" class="search" :prefix-icon="Search" clearable placeholder="搜索文章名、内容或作者" @keyup.enter="page.page = 1; search()" />
      <el-button type="primary" :icon="Search" @click="page.page = 1; search()">搜索</el-button><span class="result-count">{{ page.total }} 篇公开文章</span>
    </section>
    <section v-loading="loading" class="post-gallery">
      <article v-for="post in page.items" :key="post.id" class="post-card" @click="open(post)">
        <img v-if="post.coverImageUrl" :src="post.coverImageUrl" :alt="post.title" />
        <div><span class="post-meta">{{ post.category?.name || '未分类' }} · {{ post.authorName }}</span><h2>{{ post.title }}</h2><p>{{ post.excerpt || post.content.slice(0, 100) }}</p><footer><span><el-icon><Star /></el-icon>{{ post.ratingAverage.toFixed(1) }} ({{ post.ratingCount }})</span><span><el-icon><ChatDotRound /></el-icon>{{ post.commentCount }}</span></footer></div>
      </article>
      <el-empty v-if="!page.items.length" description="没有找到匹配的博客" />
    </section>
    <el-pagination v-if="page.total > page.pageSize" v-model:current-page="page.page" :page-size="page.pageSize" :total="page.total" layout="prev, pager, next" @current-change="search" />
    <el-dialog v-model="selected" class="reader-dialog" width="min(900px, 94vw)" destroy-on-close>
      <template #header><div v-if="selected" class="reader-heading"><span>{{ selected.authorName }}</span><h1>{{ selected.title }}</h1><p>{{ selected.excerpt }}</p></div></template>
      <template v-if="selected">
        <MdPreview :id="`post-${selected.id}`" :model-value="selected.content" preview-theme="github" />
        <div class="reader-rating"><span>读者评分 {{ selected.ratingAverage.toFixed(1) }} / 5 · {{ selected.ratingCount }} 人</span><el-tooltip :content="ratingTooltip"><span><el-rate v-model="selected.myRating" size="large" :disabled="ratingSubmitting || !canRate" @change="rate" /></span></el-tooltip></div>
        <section class="reader-comments">
          <h2>评论 {{ comments.length }}</h2>
          <div class="comment-composer"><el-input ref="commentInput" v-model="comment" type="textarea" :rows="2" maxlength="1000" show-word-limit placeholder="参与讨论" /><div class="comment-actions"><EmojiPicker @select="insertEmoji" /><el-button type="primary" :loading="sending" @click="send">评论</el-button></div></div>
          <article v-for="item in comments" :key="item.id"><strong>{{ item.authorName }}</strong><p>{{ item.content }}</p></article>
        </section>
      </template>
    </el-dialog>
  </div>
</template>
