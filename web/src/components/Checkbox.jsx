import { Check } from 'lucide-react'
import { useId } from 'react'

export default function Checkbox({ label, className = '', id, disabled = false, ...inputProps }) {
  const generatedId = useId()
  const inputId = id ?? generatedId

  return (
    <label className={`theme-checkbox${disabled ? ' theme-checkbox--disabled' : ''}${className ? ` ${className}` : ''}`} htmlFor={inputId}>
      <input {...inputProps} id={inputId} type="checkbox" disabled={disabled} />
      <span className="theme-checkbox__box" aria-hidden="true"><Check size={14} strokeWidth={3} /></span>
      <span className="theme-checkbox__label">{label}</span>
    </label>
  )
}
