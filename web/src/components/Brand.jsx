import logo from '../assets/svg/logo.svg'

export default function Brand({ onClick }) {
  return (
    <button className="brand" onClick={onClick} aria-label="见山首页">
      <img className="brand-logo" src={logo} alt="" aria-hidden="true" />
      <span className="brand-copy">
        <span className="brand-name">见山<span className="brand-dot">.</span></span>
        <small className="brand-tagline" aria-hidden="true">
          <span className="brand-tagline-item brand-tagline-english">JIAN SHAN</span>
          <span className="brand-tagline-item brand-tagline-chinese">慢慢来，合适的人值得认真遇见</span>
        </small>
      </span>
    </button>
  )
}
