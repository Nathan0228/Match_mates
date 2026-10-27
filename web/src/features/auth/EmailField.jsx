import mailIcon from '../../assets/svg/mail.svg'
import AuthField from './AuthField'

export const CAMPUS_DOMAIN = '@mails.ucas.ac.cn'

export function fullCampusEmail(localPart) {
  return `${localPart.trim()}${CAMPUS_DOMAIN}`
}

export default function EmailField({ value, onChange, disabled }) {
  const update = event => {
    const entered = event.target.value.trim()
    const localPart = entered.toLowerCase().endsWith(CAMPUS_DOMAIN) ? entered.slice(0, -CAMPUS_DOMAIN.length) : entered
    onChange(localPart)
  }

  return <AuthField
    label="国科大邮箱"
    icon={mailIcon}
    suffix={CAMPUS_DOMAIN}
    required
    type="text"
    inputMode="email"
    autoComplete="username"
    pattern="[A-Za-z0-9._%+-]+"
    title="只需填写 @ 前面的邮箱用户名"
    value={value}
    onChange={update}
    placeholder="邮箱用户名"
    disabled={disabled}
    aria-label="国科大邮箱用户名"
  />
}
