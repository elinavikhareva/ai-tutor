export type LessonStatus = 'pending' | 'in_progress' | 'done' | 'needs_review'

export interface Course {
  id: number
  title: string
  goal_depth: number
  created_at: string
}

export interface LessonSummary {
  id: number
  position: number
  title: string
  objective: string
  status: LessonStatus
}

export interface Chapter {
  id: number
  number: string
  title: string
  position: number
  status: LessonStatus
  lessons: LessonSummary[]
}

export interface CourseDetail extends Course {
  chapters: Chapter[]
}

export interface Lesson extends LessonSummary {
  chapter_id: number
  content?: string
}

export interface ChatMessage {
  role: 'user' | 'assistant'
  text: string
}
