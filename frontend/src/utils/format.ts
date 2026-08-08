import type { PostStatus } from '@/types'

export function statusLabel(status: PostStatus) {
  return ({ draft: '草稿', published: '已发布', archived: '已归档' } as const)[status]
}

export function formatDate(value?: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}
