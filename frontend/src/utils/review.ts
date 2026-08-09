import type { ReviewNotification, ReviewStatus } from '@/types'

export function reviewStatusLabel(status: ReviewStatus) {
  return ({ draft: '草稿', pending: '审核中', approved: '已通过', rejected: '已驳回', canceled: '已撤回' } as const)[status]
}

export function reviewStatusType(status: ReviewStatus): 'success' | 'warning' | 'danger' | 'info' {
  return ({ draft: 'info', pending: 'warning', approved: 'success', rejected: 'danger', canceled: 'info' } as const)[status]
}

export function reviewNotificationMessage(notification: ReviewNotification) {
  if (notification.reviewStatus === 'approved') return `《${notification.title}》审核已通过`
  if (notification.reviewStatus === 'rejected') return `《${notification.title}》已驳回${notification.reviewNote ? `：${notification.reviewNote}` : ''}`
  return `《${notification.title}》审核状态更新为${reviewStatusLabel(notification.reviewStatus)}`
}
