<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Medal } from '@element-plus/icons-vue'
import { apiMessage, blogApi } from '@/api'
import type { LeaderboardEntry } from '@/types'
const loading = ref(true); const items = ref<LeaderboardEntry[]>([])
onMounted(async () => { try { items.value = await blogApi.leaderboard() } catch (error) { ElMessage.error(apiMessage(error, '排行榜加载失败')) } finally { loading.value = false } })
</script>
<template><div class="page-stack"><section class="data-section leaderboard-summary"><el-icon><Medal /></el-icon><div><h2>博客积分排行榜</h2><p>积分综合已审核文章数、收到的评分、平均分和评论热度实时计算</p></div></section><section class="data-section table-section"><el-table v-loading="loading" :data="items" empty-text="还没有已审核文章"><el-table-column label="排名" width="80"><template #default="scope"><strong class="rank-number">{{ scope.row.rank }}</strong></template></el-table-column><el-table-column label="作者" prop="authorName" min-width="180" /><el-table-column label="积分" width="120"><template #default="scope"><strong>{{ scope.row.score }}</strong></template></el-table-column><el-table-column label="已发布" prop="publishedPosts" width="100" /><el-table-column label="平均评分" width="120"><template #default="scope">{{ scope.row.averageRating.toFixed(1) }} / 5</template></el-table-column><el-table-column label="评分数" prop="ratingCount" width="90" /><el-table-column label="评论数" prop="commentCount" width="90" /></el-table></section></div></template>
