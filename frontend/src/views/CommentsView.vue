<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Promotion } from '@element-plus/icons-vue'
import { apiMessage, blogApi } from '@/api'
import { useAuthStore } from '@/stores/auth'
import type { Comment, Post } from '@/types'
import { formatDate } from '@/utils/format'
const route = useRoute(); const auth = useAuthStore(); const posts = ref<Post[]>([]); const postId = ref(String(route.query.postId || '')); const comments = ref<Comment[]>([]); const content = ref(''); const loading = ref(false); const sending = ref(false)
async function loadComments() { if (!postId.value) { comments.value = []; return }; loading.value = true; try { comments.value = await blogApi.comments(postId.value) } catch (error) { ElMessage.error(apiMessage(error, '评论加载失败')) } finally { loading.value = false } }
async function send() { if (!content.value.trim()) return; sending.value = true; try { await blogApi.createComment(postId.value, content.value); content.value = ''; ElMessage.success('评论已添加'); await loadComments() } catch (error) { ElMessage.error(apiMessage(error, '评论失败')) } finally { sending.value = false } }
async function remove(item: Comment) { try { await ElMessageBox.confirm('确认删除这条评论？', '删除评论', { type: 'warning' }); await blogApi.deleteComment(item.id); await loadComments() } catch (error) { if (error !== 'cancel' && error !== 'close') ElMessage.error(apiMessage(error, '删除失败')) } }
watch(postId, loadComments)
onMounted(async () => { try { posts.value = (await blogApi.posts({ scope: auth.isAdmin ? 'all' : '', page: 1, pageSize: 100 })).items; if (!postId.value) postId.value = posts.value[0]?.id || ''; await loadComments() } catch (error) { ElMessage.error(apiMessage(error, '文章加载失败')) } })
</script>
<template><div class="comments-layout"><aside class="post-selector"><header><h2>选择文章</h2><span>{{ posts.length }} 篇</span></header><button v-for="post in posts" :key="post.id" :class="{ active: post.id === postId }" @click="postId = post.id"><strong>{{ post.title }}</strong><small>{{ post.commentCount }} 条评论</small></button><el-empty v-if="!posts.length" :image-size="52" description="还没有文章" /></aside><section class="comment-workspace"><div v-if="postId" class="comment-composer"><el-input v-model="content" type="textarea" :rows="3" maxlength="1000" show-word-limit placeholder="补充讨论或反馈" /><el-button type="primary" :icon="Promotion" :loading="sending" @click="send">发表评论</el-button></div><div v-loading="loading" class="comment-list"><article v-for="item in comments" :key="item.id"><span class="avatar small">{{ item.authorName.slice(0, 1) }}</span><div><header><strong>{{ item.authorName }}</strong><small>{{ formatDate(item.createdAt) }}</small></header><p>{{ item.content }}</p></div><el-tooltip v-if="item.canDelete" content="删除评论"><el-button text type="danger" :icon="Delete" aria-label="删除评论" @click="remove(item)" /></el-tooltip></article><el-empty v-if="postId && !comments.length" :image-size="64" description="还没有评论" /><el-empty v-if="!postId" :image-size="64" description="请先创建文章" /></div></section></div></template>
