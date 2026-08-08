<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { ArrowLeft, Check, Picture, Promotion } from '@element-plus/icons-vue'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { apiMessage, blogApi } from '@/api'
import type { Category, PostInput, PostStatus } from '@/types'

const route = useRoute(); const router = useRouter(); const id = computed(() => String(route.params.id || ''))
const loading = ref(Boolean(id.value)); const saving = ref(false); const categories = ref<Category[]>([])
const form = reactive<PostInput>({ title: '', slug: '', excerpt: '', content: '', coverImageUrl: '', status: 'draft', categoryId: '', tags: [] })

async function load() { categories.value = await blogApi.categories(); if (id.value) Object.assign(form, await blogApi.post(id.value)) }
async function save(status: PostStatus) {
  if (!form.title.trim() || !form.content.trim()) { ElMessage.warning('标题和正文不能为空'); return }
  saving.value = true
  try {
    form.status = status
    const result = id.value ? await blogApi.updatePost(id.value, form) : await blogApi.createPost(form)
    ElMessage.success(status === 'published' && result.reviewStatus === 'pending' ? '已提交审核' : status === 'published' ? '文章已发布' : '草稿已保存')
    await router.replace('/posts')
  } catch (error) { ElMessage.error(apiMessage(error, '保存失败')) } finally { saving.value = false }
}
async function uploadCover(options: UploadRequestOptions) {
  try { const result = await blogApi.upload(options.file); form.coverImageUrl = result.url; options.onSuccess(result); ElMessage.success('封面上传成功') }
  catch (error) { ElMessage.error(apiMessage(error, '上传失败')); throw error }
}
async function uploadContentImages(files: File[], callback: (urls: string[]) => void) {
  try { callback(await Promise.all(files.map(async (file) => (await blogApi.upload(file)).url))); ElMessage.success('图片已插入正文') }
  catch (error) { ElMessage.error(apiMessage(error, '图片上传失败')) }
}
onMounted(async () => { try { await load() } catch (error) { ElMessage.error(apiMessage(error, '文章加载失败')) } finally { loading.value = false } })
</script>

<template>
  <div v-loading="loading" class="editor-page">
    <header class="editor-toolbar">
      <el-button text :icon="ArrowLeft" aria-label="返回文章列表" @click="router.push('/posts')" />
      <div><strong>{{ id ? '编辑文章' : '新建文章' }}</strong><small>Markdown 实时预览</small></div><span class="toolbar-spacer" />
      <el-button :icon="Check" :loading="saving" @click="save('draft')">保存草稿</el-button>
      <el-button type="primary" :icon="Promotion" :loading="saving" aria-label="提交审核" @click="save('published')">提交审核</el-button>
    </header>
    <div class="editor-layout">
      <section class="editor-main">
        <el-input v-model="form.title" class="title-input" maxlength="180" placeholder="文章标题" />
        <el-input v-model="form.excerpt" type="textarea" :rows="2" maxlength="500" show-word-limit placeholder="摘要，帮助读者快速判断内容" />
        <MdEditor v-model="form.content" class="content-editor rich-markdown-editor" language="zh-CN" preview-theme="github" :toolbars-exclude="['github']" @on-upload-img="uploadContentImages" />
      </section>
      <aside class="editor-settings">
        <section><h2>发布设置</h2><el-form label-position="top"><el-form-item label="URL 标识"><el-input v-model="form.slug" placeholder="留空时根据标题生成" /></el-form-item><el-form-item label="分类"><el-select v-model="form.categoryId" clearable placeholder="未分类"><el-option v-for="item in categories" :key="item.id" :label="item.name" :value="item.id" /></el-select></el-form-item><el-form-item label="标签"><el-select v-model="form.tags" multiple filterable allow-create default-first-option :multiple-limit="10" placeholder="输入后回车" /></el-form-item></el-form></section>
        <section><h2>封面图片</h2><el-upload class="cover-upload" :show-file-list="false" accept="image/jpeg,image/png,image/gif,image/webp" :http-request="uploadCover"><img v-if="form.coverImageUrl" :src="form.coverImageUrl" alt="文章封面" /><div v-else><el-icon><Picture /></el-icon><span>选择图片</span><small>最大 5 MiB</small></div></el-upload><el-button v-if="form.coverImageUrl" link type="danger" @click="form.coverImageUrl = ''">移除封面</el-button></section>
      </aside>
    </div>
  </div>
</template>
