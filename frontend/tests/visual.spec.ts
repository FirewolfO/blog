import { expect, test, type Page } from '@playwright/test'

const posts = [
  { id: 'p1', title: 'Gateway 路由治理实践', slug: 'gateway-routing', excerpt: '从认证、匹配到可观测性的完整实践。', content: '# Gateway 路由治理\n\n这是一篇用于视觉检查的 Markdown 文章。', coverImageUrl: '', status: 'published', reviewStatus: 'approved', reviewNote: '', hasPendingChanges: false, categoryId: '', tags: ['Go', 'Gateway'], authorId: 'u1', authorName: '林工程师', publishedAt: '2026-08-08T10:00:00Z', createdAt: '2026-08-08T10:00:00Z', updatedAt: '2026-08-08T10:00:00Z', commentCount: 8, ratingCount: 12, ratingAverage: 4.8, myRating: 0, canEdit: false, canDelete: false, recommendationReason: '与你关注的 2 个标签相关' },
  { id: 'p2', title: 'SQLite 索引设计笔记', slug: 'sqlite-indexes', excerpt: '用查询计划定位索引设计问题。', content: '# SQLite 索引设计', coverImageUrl: '', status: 'published', reviewStatus: 'approved', reviewNote: '', hasPendingChanges: false, categoryId: '', tags: ['SQLite'], authorId: 'u2', authorName: '周同学', publishedAt: '2026-08-07T10:00:00Z', createdAt: '2026-08-07T10:00:00Z', updatedAt: '2026-08-07T10:00:00Z', commentCount: 3, ratingCount: 7, ratingAverage: 4.6, myRating: 0, canEdit: false, canDelete: false, recommendationReason: '读者评分 4.6' },
]
const comments = [{ id: 'c1', postId: 'p1', content: '很有帮助 👍', authorId: 'u3', authorName: '读者', createdAt: '2026-08-09T09:00:00Z', canDelete: false }]

async function mockAPI(page: Page) {
  await page.addInitScript(() => sessionStorage.setItem('blog_access_token', 'visual-test-token'))
  await page.route('**/api/v1/**', async (route) => {
    const url = new URL(route.request().url()); let data: unknown
    if (url.pathname.endsWith('/auth/me')) data = { user: { id: 'viewer', username: 'viewer', displayName: '视觉检查用户', permissions: [] } }
    else if (url.pathname.endsWith('/recommendations')) data = posts
    else if (/\/posts\/[^/]+\/comments$/.test(url.pathname)) data = comments
    else if (/\/posts\/[^/]+\/rating$/.test(url.pathname)) data = { stars: 5, ratingCount: 13, ratingAverage: 4.8 }
    else if (/\/posts\/[^/]+$/.test(url.pathname)) data = posts.find((post) => url.pathname.endsWith(`/posts/${post.id}`))
    else if (url.pathname.endsWith('/posts')) data = { items: posts, total: posts.length, page: 1, pageSize: 12 }
    else if (url.pathname.endsWith('/categories')) data = [{ id: 'cat1', name: '工程实践', slug: 'engineering', description: '', postCount: 2 }]
    else data = {}
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify({ code: 'OK', message: 'success', data, requestId: 'visual' }) })
  })
}

test('explore page is usable on desktop', async ({ page }) => {
  await mockAPI(page); await page.setViewportSize({ width: 1440, height: 900 }); await page.goto('http://127.0.0.1:5179/explore')
  await expect(page.getByRole('heading', { name: '发现博客' })).toBeVisible(); await expect(page.getByText('Gateway 路由治理实践').first()).toBeVisible()
  await page.locator('.post-card').first().click()
  await expect(page.locator('.reader-rating .el-rate__item')).toHaveCount(5)
  await page.getByRole('button', { name: '添加表情' }).click()
  await page.getByRole('button', { name: '插入 🎉' }).click()
  await expect(page.getByPlaceholder('参与讨论')).toHaveValue('🎉')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
  await page.screenshot({ path: '/tmp/blog-interactions-desktop.png', fullPage: true })
})

test('comment tools fit on mobile', async ({ page }) => {
  await mockAPI(page); await page.setViewportSize({ width: 390, height: 844 }); await page.goto('http://127.0.0.1:5179/comments')
  await expect(page.getByPlaceholder('补充讨论或反馈')).toBeVisible()
  await expect(page.locator('.comment-rating .el-rate__item')).toHaveCount(5)
  await page.getByRole('button', { name: '添加表情' }).click()
  await expect(page.getByRole('button', { name: '插入 👍' })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
  await page.screenshot({ path: '/tmp/blog-comments-mobile.png', fullPage: true })
})

test('markdown editor fits on mobile', async ({ page }) => {
  await mockAPI(page); await page.setViewportSize({ width: 390, height: 844 }); await page.goto('http://127.0.0.1:5179/posts/new')
  await expect(page.getByPlaceholder('文章标题')).toBeVisible(); await expect(page.getByRole('button', { name: '提交审核' })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
  await page.screenshot({ path: '/tmp/task7-editor-mobile.png', fullPage: true })
})
