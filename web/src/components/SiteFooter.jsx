import Brand from './Brand'

export default function SiteFooter() {
  return (
    <footer>
      <div className="wrap footer-inner">
        <Brand onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })} />
        <p>在山水之间，遇见彼此。</p>
        <span>非国科大官方项目 · 仅面向国科大学生</span>
        <span>© 2026 见山</span>
        <div className="tmdb-credit"><a href="https://www.themoviedb.org" target="_blank" rel="noreferrer" aria-label="TMDB"><img src="https://www.themoviedb.org/assets/2/v4/logos/v2/blue_square_1-5bdc75aaebeb75dc7ae79426ddd9be3b2be1e342510f8202baf6bffa71d7f5c4.svg" alt="TMDB" loading="lazy" /></a><small>This product uses the TMDB API but is not endorsed or certified by TMDB.</small></div>
      </div>
    </footer>
  )
}
