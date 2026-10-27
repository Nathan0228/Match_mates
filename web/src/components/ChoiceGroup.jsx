import { Check, ChevronDown } from 'lucide-react'
import { useId, useRef } from 'react'

export default function ChoiceGroup({ label, hint, options, value, onChange, emptyValue = '', layout = 'list', disabled = false, required = false, collapseOnSelect = false }) {
  const id = useId()
  const detailsRef = useRef(null)
  const summaryRef = useRef(null)
  const hasSelection = value !== emptyValue
  const selectedLabel = options.find(option => option.value === value)?.label
  const select = optionValue => {
    onChange(optionValue)
    if (collapseOnSelect && detailsRef.current) {
      detailsRef.current.open = false
      summaryRef.current?.focus()
    }
  }

  return (
    <details ref={detailsRef} className={`choice-group choice-group--${layout}${disabled ? ' choice-group--disabled' : ''}`} onInvalidCapture={() => { if (detailsRef.current) detailsRef.current.open = true }}>
      <summary ref={summaryRef} className="choice-group__toggle">
        <span className="choice-group__title">{label}</span>
        {hasSelection && selectedLabel && <span className="choice-group__selection">{selectedLabel}</span>}
        <ChevronDown className="choice-group__chevron" size={17} aria-hidden="true" />
      </summary>
      <div className="choice-group__body">
        <fieldset className="choice-group__fieldset" disabled={disabled} aria-describedby={hint ? `${id}-hint` : undefined}>
          <legend className="choice-group__legend">{label}</legend>
          {(hint || (!required && !disabled && hasSelection)) && <div className="choice-group__meta">
            {hint && <p id={`${id}-hint`}>{hint}</p>}
            {!required && !disabled && hasSelection && <button type="button" className="choice-group__clear" onClick={() => onChange(emptyValue)}>清除选择</button>}
          </div>}
          <div className="choice-group__options">
            {options.map(option => (
              <label key={option.value} className={`choice-group__option${value === option.value ? ' is-selected' : ''}`}>
                <input type="radio" name={id} value={option.value} checked={value === option.value} onChange={() => select(option.value)} required={required} aria-label={option.ariaLabel} />
                <span className="choice-group__option-text">{option.label}</span>
                {layout !== 'scale' && <span className="choice-group__mark" aria-hidden="true"><Check size={13} strokeWidth={3} /></span>}
              </label>
            ))}
          </div>
        </fieldset>
      </div>
    </details>
  )
}
