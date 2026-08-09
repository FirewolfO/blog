<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Close, EditPen } from '@element-plus/icons-vue'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import { apiMessage, blogApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { Post, ReviewSubmission } from '@/types'
import { formatDate } from '@/utils/format'
import { reviewStatusLabel, reviewStatusType } from '@/utils/review'

const router = useRouter()
const auth = useAuthStore()
const activeTab = ref('mine')
const loading = ref(true)
const mine = ref<ReviewSubmission[]>([])
const queue = ref<Post[]>([])
const selected = ref<Post>()
const pendingMine = computed(() => mine.value.filter((item) => item.reviewStatus === 'pending').length)

function reviewOutcome(item: ReviewSubmission) {
  if (item.reviewStatus === 'rejected') return item.reviewNote || '未填写驳回原因'
  if (item.reviewStatus === 'approved') return '审核通过'
  if (item.reviewStatus === 'canceled') return '投稿已撤回'
  return '等待审核'
}

async function load() {
  loading.value = true
  try {
    const [myItems, queueItems] = await Promise.all([
      blogApi.myReviews(),
      auth.canReview ? blogApi.reviews() : Promise.resolve([]),
    ])
    mine.value = myItems
    queue.value = queueItems
    if (selected.value && !queueItems.some((item) => item.id === selected.value?.id)) selected.value = undefined
  } catch (error) {
    ElMessage.error(apiMessage(error, '审核记录加载失败'))
  } finally {
    loading.value = false
  }
}

async function approve(post: Post) {
  try {
    await ElMessageBox.confirm(`确认通过《${post.title}》？`, '审核通过')
    await blogApi.approve(post.id)
    ElMessage.success('已通过审核')
    selected.value = undefined
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') ElMessage.error(apiMessage(error, '审核失败'))
  }
}

async function reject(post: Post) {
  try {
    const result = await ElMessageBox.prompt('请填写作者可见的驳回原因', `驳回《${post.title}》`, { inputPattern: /\S+/, inputErrorMessage: '驳回原因不能为空' })
    await blogApi.reject(post.id, result.value)
    ElMessage.success('已驳回')
    selected.value = undefined
    await load()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') ElMessage.error(apiMessage(error, '审核失败'))
  }
}

onMounted(async () => {
  await load()
  try { await blogApi.readReviewNotifications() } catch { /* The list remains usable if marking notifications fails. */ }
})
</script>

<template>
  <div class="review-page">
    <el-tabs v-model="activeTab" class="review-tabs">
      <el-tab-pane name="mine">
        <template #label><span>我的提交（{{ mine.length }}）</span></template>
        <section v-loading="loading" class="review-history">
          <header><div><h2>我的审核记录</h2><p>查看提交状态、审核结果和处理原因</p></div><el-tag v-if="pendingMine" type="warning" effect="plain">{{ pendingMine }} 条审核中</el-tag></header>
          <el-table class="review-history-table" :data="mine" empty-text="还没有发起过审核">
            <el-table-column label="文章" min-width="220"><template #default="scope"><div class="review-title"><strong>{{ scope.row.title }}</strong><small>{{ scope.row.submissionType === 'revision' ? '修改版本' : '首次发布' }}</small></div></template></el-table-column>
            <el-table-column label="提交时间" width="170"><template #default="scope">{{ formatDate(scope.row.submittedAt) }}</template></el-table-column>
            <el-table-column label="状态" width="110"><template #default="scope"><el-tag :type="reviewStatusType(scope.row.reviewStatus)" effect="plain">{{ reviewStatusLabel(scope.row.reviewStatus) }}</el-tag></template></el-table-column>
            <el-table-column label="审核结果 / 原因" min-width="240"><template #default="scope"><span class="review-outcome" :class="{ rejected: scope.row.reviewStatus === 'rejected' }">{{ reviewOutcome(scope.row) }}</span><small v-if="scope.row.reviewedAt" class="reviewed-at">{{ formatDate(scope.row.reviewedAt) }}</small></template></el-table-column>
            <el-table-column label="操作" width="100" fixed="right"><template #default="scope"><el-button v-if="scope.row.reviewStatus === 'rejected'" link type="primary" :icon="EditPen" @click="router.push(`/posts/${scope.row.postId}/edit`)">修改</el-button></template></el-table-column>
          </el-table>
          <div class="review-history-mobile">
            <article v-for="item in mine" :key="item.id">
              <header><div class="review-title"><strong>{{ item.title }}</strong><small>{{ item.submissionType === 'revision' ? '修改版本' : '首次发布' }} · {{ formatDate(item.submittedAt) }}</small></div><el-tag :type="reviewStatusType(item.reviewStatus)" effect="plain">{{ reviewStatusLabel(item.reviewStatus) }}</el-tag></header>
              <p class="review-outcome" :class="{ rejected: item.reviewStatus === 'rejected' }">{{ reviewOutcome(item) }}</p>
              <footer><small v-if="item.reviewedAt" class="reviewed-at">处理于 {{ formatDate(item.reviewedAt) }}</small><span v-else></span><el-button v-if="item.reviewStatus === 'rejected'" link type="primary" :icon="EditPen" @click="router.push(`/posts/${item.postId}/edit`)">修改</el-button></footer>
            </article>
            <el-empty v-if="!mine.length" :image-size="64" description="还没有发起过审核" />
          </div>
        </section>
      </el-tab-pane>
      <el-tab-pane v-if="auth.canReview" name="queue">
        <template #label><span>待我审核（{{ queue.length }}）</span></template>
        <div class="review-layout">
          <section v-loading="loading" class="review-queue"><header><h2>待审核</h2><span>{{ queue.length }}</span></header><button v-for="post in queue" :key="post.id" :class="{ active: selected?.id === post.id }" @click="selected = post"><strong>{{ post.title }}</strong><small>{{ post.authorName }} · {{ post.hasPendingChanges ? '修改版本' : '新文章' }}</small></button><el-empty v-if="!queue.length" :image-size="72" description="暂无待审内容" /></section>
          <section class="review-preview"><template v-if="selected"><header><div><span>{{ selected.authorName }}</span><h1>{{ selected.title }}</h1></div><el-button :icon="Close" @click="reject(selected)">驳回</el-button><el-button type="primary" :icon="Check" @click="approve(selected)">通过</el-button></header><p class="review-excerpt">{{ selected.excerpt }}</p><MdPreview :id="`review-${selected.id}`" :model-value="selected.content" preview-theme="github" /></template><el-empty v-else description="选择一篇内容开始审核" /></section>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>
