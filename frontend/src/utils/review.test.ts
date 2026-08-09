import { describe, expect, it } from 'vitest'
import { reviewNotificationMessage, reviewStatusLabel, reviewStatusType } from './review'

describe('review presentation', () => {
  it('formats every author-visible review status', () => {
    expect(reviewStatusLabel('pending')).toBe('审核中')
    expect(reviewStatusLabel('approved')).toBe('已通过')
    expect(reviewStatusLabel('rejected')).toBe('已驳回')
    expect(reviewStatusLabel('canceled')).toBe('已撤回')
    expect(reviewStatusType('rejected')).toBe('danger')
  })

  it('includes the rejection reason in result notifications', () => {
    expect(reviewNotificationMessage({ id: 'n1', reviewSubmissionId: 'r1', postId: 'p1', title: '迁移方案', reviewStatus: 'rejected', reviewNote: '请补充回滚步骤', createdAt: '2026-08-09T00:00:00Z' })).toBe('《迁移方案》已驳回：请补充回滚步骤')
  })
})
