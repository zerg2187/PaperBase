import { useState, useEffect } from 'react'
import type { SearchResult, Tag } from '../types'

const CART_STORAGE_KEY = 'paperbase_cart'

// ロール切替（ログイン/ログアウト）時に呼ぶ。カートは sessionStorage に永続化される
// ため、リロードだけでは前ロールの論文（bibtex 込み）が持ち越されてしまう
export function clearCartStorage() {
  sessionStorage.removeItem(CART_STORAGE_KEY)
}

export interface CartItem {
  id: string
  title: string
  authors: string[]
  year: number
  bibtex: string
  tags?: Tag[]
}

interface UseCartReturn {
  cart: CartItem[]
  addToCart: (paper: SearchResult) => void
  removeFromCart: (id: string) => void
  clearCart: () => void
  isInCart: (id: string) => boolean
}

export function useCart(): UseCartReturn {
  const [cart, setCart] = useState<CartItem[]>([])

  // Load cart from sessionStorage on mount
  useEffect(() => {
    try {
      const stored = sessionStorage.getItem(CART_STORAGE_KEY)
      if (stored) {
        setCart(JSON.parse(stored))
      }
    } catch (e) {
      console.error('Failed to load cart from sessionStorage:', e)
    }
  }, [])

  // Save cart to sessionStorage whenever it changes
  useEffect(() => {
    try {
      sessionStorage.setItem(CART_STORAGE_KEY, JSON.stringify(cart))
    } catch (e) {
      console.error('Failed to save cart to sessionStorage:', e)
    }
  }, [cart])

  const addToCart = (paper: SearchResult) => {
    setCart(prev => {
      if (prev.some(item => item.id === paper.id)) {
        return prev
      }
      return [...prev, {
        id: paper.id,
        title: paper.title,
        authors: paper.authors,
        year: paper.year,
        bibtex: paper.bibtex,
        tags: paper.tags,
      }]
    })
  }

  const removeFromCart = (id: string) => {
    setCart(prev => prev.filter(item => item.id !== id))
  }

  const clearCart = () => {
    setCart([])
  }

  const isInCart = (id: string): boolean => {
    return cart.some(item => item.id === id)
  }

  return {
    cart,
    addToCart,
    removeFromCart,
    clearCart,
    isInCart,
  }
}
