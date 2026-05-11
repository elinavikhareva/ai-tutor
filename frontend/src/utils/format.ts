const DEPTH_LABELS: Record<number, string> = {
  1: 'поверхностно',
  2: 'рабочий уровень',
  3: 'эксперт',
}

export function depthLabel(depth: number): string {
  return DEPTH_LABELS[depth] ?? DEPTH_LABELS[2]!
}

const pluralRules = new Intl.PluralRules('ru')

// plural(5, ['урок', 'урока', 'уроков']) === 'уроков'
export function plural(n: number, [one, few, many]: [string, string, string]): string {
  switch (pluralRules.select(n)) {
    case 'one':
      return one
    case 'few':
      return few
    default:
      return many
  }
}

const ACCENTS = ['#0E93AD', '#4254CC', '#0F9D5E', '#BC7A14', '#CE3155', '#9340D0']

// Courses have no categories, so the accent colour is just derived from the id
// to tell cards apart.
export function courseAccent(courseId: number): string {
  return ACCENTS[courseId % ACCENTS.length]!
}
