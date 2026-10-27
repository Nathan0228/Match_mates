import { contacts, paces, rhythms, values } from '../../data/preferences'
import ChoiceGroup from '../../components/ChoiceGroup'

const energyOptions = [1, 2, 3, 4, 5].map(value => ({
  value,
  label: String(value),
  ariaLabel: `${value}，${value === 1 ? '更喜欢独处' : value === 5 ? '更喜欢社交' : '独处与社交之间'}`,
}))

export default function MatchPreferencesEditor({ preferences, onChange }) {
  const set = (key, value) => onChange({ ...preferences, [key]: value })
  const toggleValue = value => {
    const current = preferences.values || []
    if (current.includes(value)) set('values', current.filter(item => item !== value))
    else if (current.length < 3) set('values', [...current, value])
  }

  return <section className="preference-section" aria-label="相处偏好">
    <h3>更容易聊得来的线索</h3>
    <p className="field-hint">这些内容会展示给符合交友方向的同学，可慢慢补充。匹配分只用于排序，不预测两个人会不会互相喜欢。</p>
    <label>相处中最看重什么？ <small>最多选 3 项</small></label>
    <div className="interest-list">{values.map(value => <button type="button" className={preferences.values?.includes(value) ? 'picked' : ''} disabled={(preferences.values?.length || 0) >= 3 && !preferences.values.includes(value)} key={value} onClick={() => toggleValue(value)}>{value}</button>)}</div>
    <div className="choice-grid">
      <ChoiceGroup label="认识的节奏" hint="你希望怎样开始了解彼此？" options={paces} value={preferences.pace || ''} onChange={value => set('pace', value)} />
      <ChoiceGroup label="联系的习惯" hint="选一个最舒服的交流频率" options={contacts} value={preferences.contact || ''} onChange={value => set('contact', value)} />
      <ChoiceGroup label="社交充电方式" hint="1 更喜欢独处 · 5 更喜欢社交" options={energyOptions} value={preferences.energy || 0} emptyValue={0} layout="scale" onChange={value => set('energy', value)} />
      <ChoiceGroup label="平常作息" hint="日常更接近哪种状态？" options={rhythms} value={preferences.rhythm || ''} onChange={value => set('rhythm', value)} />
    </div>
    <label>MBTI <small>仅作聊天话题，不参与匹配分</small><input maxLength="4" pattern="[EI][SN][TF][JP]" title="请输入如 INFP 的四位类型，或留空" value={preferences.mbti || ''} onChange={event => set('mbti', event.target.value.toUpperCase())} placeholder="例如 INFP；不了解可留空" /></label>
    <div className="field-pair">
      <label>一个普通周末，我会… <small>可选，最多 120 字</small><textarea maxLength="120" rows="3" value={preferences.daily || ''} onChange={event => set('daily', event.target.value)} placeholder="比如去湖边散步，或做一顿饭" /></label>
      <label>如果我们认识，想一起… <small>可选，最多 120 字</small><textarea maxLength="120" rows="3" value={preferences.together || ''} onChange={event => set('together', event.target.value)} placeholder="比如看展、打球、分享最近读的书" /></label>
    </div>
  </section>
}
