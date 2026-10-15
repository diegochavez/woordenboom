import {
  Bank, BookStack, Bus, Cart, Check, Clock, Cloud, Community, Computer,
  Flash, GraduationCap, HealthShield, HomeSimple, Learning, ListSelect,
  Map as MapIcon, NavArrowLeft, NavArrowRight, Page, Phone, Restart,
  Settings, Shield, Suitcase, Tools, Xmark, ThumbsDown, ThumbsUp,
} from '@/lib/iconoir';
import { Platform, View } from 'react-native';
import { colors } from '@/lib/theme';

const icons = {
  'arrow-left': NavArrowLeft, book: BookStack, check: Check, 'chevron-right': NavArrowRight,
  flip: Restart, map: MapIcon, 'thumbs-down': ThumbsDown, 'thumbs-up': ThumbsUp, x: Xmark,
  learning: Learning, categories: ListSelect, clock: Clock, home: HomeSimple,
  health: HealthShield, food: Cart, transport: Bus, work: Suitcase, bank: Bank,
  school: GraduationCap, repair: Tools, computer: Computer, government: Bank,
  documents: Page, legal: Shield, news: Page, utilities: Flash, emergency: Shield,
  phone: Phone, weather: Cloud, social: Community, engineering: Settings,
};
export type IconName = keyof typeof icons;
type LineIconProps = { color?: string; name: IconName; size?: number; strokeWidth?: number };
export function LineIcon({ color = colors.ink, name, size = 24, strokeWidth = 1.5 }: LineIconProps) {
  const Icon = icons[name];
  // The surrounding control provides its label. Never announce decorative SVGs.
  return <View accessible={false} accessibilityElementsHidden importantForAccessibility="no-hide-descendants" aria-hidden pointerEvents="none" style={{ width: size, height: size, flexShrink: 0 }}>
    <Icon color={color} width={size} height={size} strokeWidth={strokeWidth} accessible={Platform.OS === 'web' ? undefined : false} />
  </View>;
}
const topicIcons: Record<string, IconName> = {
  'general-vocabulary': 'book', 'daily-basics': 'learning', 'housing-renting': 'home',
  healthcare: 'health', 'supermarket-food': 'food', 'public-transport': 'transport',
  'work-office': 'work', 'banking-money': 'bank', 'school-childcare': 'school',
  'home-repair': 'repair', 'software-it': 'computer', 'government-municipality': 'government',
  'immigration-documents': 'documents', 'legal-contracts': 'legal', 'politics-news': 'news',
  'utilities-bills': 'utilities', emergencies: 'emergency', 'appointments-phone': 'phone',
  'weather-small-talk': 'weather', 'social-life': 'social', 'engineering-technical': 'engineering',
};
export const iconForTopic = (slug: string): IconName => topicIcons[slug] ?? 'book';
