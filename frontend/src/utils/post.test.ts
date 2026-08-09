import { describe, expect, it } from 'vitest'
import type { PostInput } from '@/types'
import { toPostInput } from './post'

describe('toPostInput', () => {
  it('removes read-only post fields from an update payload', () => {
    const input = {
      id: 'post-1',
      title: 'Updated title',
      slug: 'updated-title',
      excerpt: 'Updated excerpt',
      content: 'Updated content',
      coverImageUrl: '',
      status: 'published',
      categoryId: '',
      tags: ['release'],
      reviewStatus: 'approved',
      ratingAverage: 5,
    } as PostInput & Record<string, unknown>

    expect(toPostInput(input)).toEqual({
      title: 'Updated title',
      slug: 'updated-title',
      excerpt: 'Updated excerpt',
      content: 'Updated content',
      coverImageUrl: '',
      status: 'published',
      categoryId: '',
      tags: ['release'],
    })
  })
})
