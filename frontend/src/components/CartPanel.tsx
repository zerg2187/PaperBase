import type { CartItem } from '../hooks/useCart'

type CartPanelProps = {
  cart: CartItem[]
  onRemove: (id: string) => void
  onClear: () => void
  onExport: () => void
  onExportPresentation: () => void
  isOpen: boolean
  onToggle: () => void
}

export function CartPanel({
  cart,
  onRemove,
  onClear,
  onExport,
  onExportPresentation,
  isOpen,
  onToggle,
}: CartPanelProps) {
  return (
    <>
      <button
        onClick={onToggle}
        className={`cart-toggle-btn ${cart.length > 0 ? 'has-items' : ''}`}
        title="カートを開く"
      >
        <span className="cart-icon">🛒</span>
        {cart.length > 0 && (
          <span className="cart-badge">{cart.length}</span>
        )}
        <span>カート</span>
      </button>

      {isOpen && (
        <div className="cart-panel">
          <div className="cart-panel-header">
            <h3 className="cart-panel-title">BibTeXカート</h3>
            <button onClick={onToggle} className="cart-panel-close">×</button>
          </div>

          <div className="cart-panel-content">
            {cart.length === 0 ? (
              <p className="cart-empty">カートは空です</p>
            ) : (
              <div className="cart-items">
                {cart.map(item => (
                  <div key={item.id} className="cart-item">
                    <div className="cart-item-info">
                      <p className="cart-item-title">{item.title}</p>
                    </div>
                    <button
                      onClick={() => onRemove(item.id)}
                      className="cart-item-remove"
                      title="削除"
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            )}

            {cart.length > 0 && (
              <div className="cart-actions">
                <button onClick={onClear} className="cart-btn-clear">
                  クリア
                </button>
                <button onClick={onExport} className="cart-btn-export">
                  📥 BibTeX ({cart.length}件)
                </button>
                <button onClick={onExportPresentation} className="cart-btn-export cart-btn-presentation">
                  📊 プレゼン ({cart.length}件)
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )
}
