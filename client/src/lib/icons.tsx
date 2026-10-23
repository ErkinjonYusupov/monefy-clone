import {
  Baby, Banknote, Bath, BookOpen, Briefcase, Bus, Car, CarTaxiFront, Coffee, CreditCard, Dumbbell,
  Fuel, Gamepad2, Gift, GraduationCap, HeartPulse, House, Landmark, Laptop, Music, PawPrint,
  PiggyBank, Plane, Receipt, Shirt, ShoppingCart, Smartphone, Star, TrendingUp, Utensils,
  Wallet, Wrench, type LucideIcon,
} from 'lucide-react'

// Serverda ikonka nomi saqlanadi; bu yerda faqat ishlatiladiganlari import qilinadi.
export const ICONS: Record<string, LucideIcon> = {
  'shopping-cart': ShoppingCart,
  utensils: Utensils,
  bus: Bus,
  'car-taxi-front': CarTaxiFront,
  car: Car,
  house: House,
  receipt: Receipt,
  smartphone: Smartphone,
  shirt: Shirt,
  'heart-pulse': HeartPulse,
  bath: Bath,
  'gamepad-2': Gamepad2,
  dumbbell: Dumbbell,
  'paw-print': PawPrint,
  gift: Gift,
  briefcase: Briefcase,
  landmark: Landmark,
  'piggy-bank': PiggyBank,
  wallet: Wallet,
  'credit-card': CreditCard,
  'graduation-cap': GraduationCap,
  plane: Plane,
  baby: Baby,
  'book-open': BookOpen,
  coffee: Coffee,
  fuel: Fuel,
  music: Music,
  wrench: Wrench,
  star: Star,
  banknote: Banknote,
  'trending-up': TrendingUp,
  laptop: Laptop,
}

export const ICON_NAMES = Object.keys(ICONS)
export const ACCOUNT_ICONS = ['wallet', 'credit-card', 'banknote', 'landmark', 'piggy-bank', 'briefcase']

export const COLORS = [
  '#F44336', '#E91E63', '#9C27B0', '#673AB7', '#3F51B5', '#2196F3', '#00BCD4', '#009688',
  '#4CAF50', '#8BC34A', '#FFC107', '#FF9800', '#FF5722', '#795548', '#607D8B', '#A1887F',
]

export function Icon({ name, size = 22, color }: { name: string; size?: number; color?: string }) {
  const C = ICONS[name] ?? Star
  return <C size={size} color={color} strokeWidth={2} />
}
