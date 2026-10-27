export default function AuthField({ label, icon, suffix, ...inputProps }) {
  return (
    <label className="auth-field">
      {label}
      <span className="auth-field__control">
        <img className="auth-field__icon" src={icon} alt="" aria-hidden="true" />
        <input {...inputProps} />
        {suffix && <span className="auth-field__suffix">{suffix}</span>}
      </span>
    </label>
  )
}
