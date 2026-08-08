<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Close } from '@element-plus/icons-vue'
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/preview.css'
import { apiMessage, blogApi } from '@/api'
import type { Post } from '@/types'
const loading = ref(true); const items = ref<Post[]>([]); const selected = ref<Post>()
async function load() { loading.value = true; try { items.value = await blogApi.reviews() } catch (error) { ElMessage.error(apiMessage(error, '待审内容加载失败')) } finally { loading.value = false } }
async function approve(post: Post) { try { await ElMessageBox.confirm(`确认通过《${post.title}》？`, '审核通过'); await blogApi.approve(post.id); ElMessage.success('已通过审核'); selected.value = undefined; await load() } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error(apiMessage(error, '审核失败')) } }
async function reject(post: Post) { try { const result = await ElMessageBox.prompt('请填写作者可见的驳回原因', `驳回《${post.title}》`, { inputPattern: /\S+/, inputErrorMessage: '驳回原因不能为空' }); await blogApi.reject(post.id, result.value); ElMessage.success('已驳回'); selected.value = undefined; await load() } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error(apiMessage(error, '审核失败')) } }
onMounted(load)
</script>
<template><div class="review-layout"><section v-loading="loading" class="review-queue"><header><h2>待审核</h2><span>{{ items.length }}</span></header><button v-for="post in items" :key="post.id" :class="{ active: selected?.id === post.id }" @click="selected = post"><strong>{{ post.title }}</strong><small>{{ post.authorName }} · {{ post.hasPendingChanges ? '修改版本' : '新文章' }}</small></button><el-empty v-if="!items.length" :image-size="72" description="暂无待审内容" /></section><section class="review-preview"><template v-if="selected"><header><div><span>{{ selected.authorName }}</span><h1>{{ selected.title }}</h1></div><el-button :icon="Close" @click="reject(selected)">驳回</el-button><el-button type="primary" :icon="Check" @click="approve(selected)">通过</el-button></header><p class="review-excerpt">{{ selected.excerpt }}</p><MdPreview :id="`review-${selected.id}`" :model-value="selected.content" preview-theme="github" /></template><el-empty v-else description="选择一篇内容开始审核" /></section></div></template>
