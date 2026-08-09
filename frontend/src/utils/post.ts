import type { PostInput } from '@/types'

export function toPostInput(input: PostInput): PostInput {
  return {
    title: input.title,
    slug: input.slug,
    excerpt: input.excerpt,
    content: input.content,
    coverImageUrl: input.coverImageUrl,
    status: input.status,
    categoryId: input.categoryId || '',
    tags: [...input.tags],
  }
}
