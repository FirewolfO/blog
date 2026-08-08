import { describe, expect, it } from 'vitest'
import { statusLabel } from './format'

describe('statusLabel', () => {
  it('returns localized post status labels', () => {
    expect(statusLabel('draft')).toBe('草稿')
    expect(statusLabel('published')).toBe('已发布')
    expect(statusLabel('archived')).toBe('已归档')
  })
})
