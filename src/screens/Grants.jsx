import { useState, useEffect, useRef } from 'react';
import { useMobile } from '../hooks/useMobile';
import { useNavigate } from 'react-router-dom';
import Icon from '../components/Icon';
import TopBar from '../components/TopBar';
import Card from '../components/Card';
import Chip from '../components/Chip';
import Button from '../components/Button';
import Search from '../components/Search';
import Progress from '../components/Progress';
import { api } from '../lib/api';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';
import { calculateRealMatch } from '../utils/match';

const C = {
  ink900: '#0A1230', ink700: '#2A3457', ink500: '#5A6485', ink300: '#9AA3BF',
  blue: '#1448FF', blue50: '#F2F5FF', blue100: '#E6ECFF',
  paper: '#FFFFFF', mist: '#F5F7FB', mist2: '#ECEFF7', hairline: '#E4E8F1',
  success700: '#047857', success100: '#ECFAF3',
  warn700: '#8A5A0E', warn100: '#FFF6E5',
  font: 'var(--font-sans)',
  mono: '"Geist Mono", monospace',
};

function GrantDetailModal({ grant, onClose, t, lang, portfolio }) {
  if (!grant) return null;
  const navigate = useNavigate();
  const tags = grant.tags || ['Грант'];

  const askAI = () => {
    localStorage.setItem('talap_chat_prefill', `Расскажи подробнее про требования и процесс подачи документов на грант "${grant.name}". Как мне лучше подготовиться и написать мотивационное эссе?`);
    navigate('/chat');
  };

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', inset: 0, background: 'rgba(10,18,48,0.5)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 200, backdropFilter: 'blur(4px)', padding: 24,
      }}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{
          background: C.paper, borderRadius: 20, width: '100%', maxWidth: 640,
          maxHeight: '85vh', overflow: 'auto',
          boxShadow: '0 32px 80px rgba(10,18,48,0.2)',
        }}
      >
        <div style={{ padding: 32, borderBottom: `1px solid ${C.hairline}`, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
              {tags.map(tag => <Chip key={tag}>{tag}</Chip>)}
            </div>
            <h2 style={{ fontFamily: C.font, fontSize: 24, fontWeight: 800, letterSpacing: '-0.02em', color: C.ink900, margin: '0 0 4px' }}>
              {grant.name}
            </h2>
            <div style={{ fontFamily: C.font, fontSize: 14, color: C.ink500 }}>{grant.subtitle} · {grant.country}</div>
          </div>
          <button onClick={onClose} style={{ width: 36, height: 36, borderRadius: 8, background: C.mist, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
            <Icon name="plus" size={18} color={C.ink500} style={{ transform: 'rotate(45deg)' }} />
          </button>
        </div>

        <div style={{ padding: 32 }}>
          {(() => {
            const realMatchScore = calculateRealMatch(grant, portfolio, false);
            return (
              <div style={{ display: 'grid', gridTemplateColumns: realMatchScore ? 'repeat(3, 1fr)' : 'repeat(2, 1fr)', gap: 12, marginBottom: 28 }}>
                {[
                  { label: t('grants_amount') || 'Сумма', value: grant.amount },
                  { label: t('grants_deadline') || 'Дедлайн', value: `${grant.deadline_days} ${t('days') || 'дн.'}`, tone: grant.tone === 'warn' ? 'warn' : null },
                  realMatchScore ? { label: 'Match', value: `${realMatchScore}%`, tone: 'success' } : null,
                ].filter(Boolean).map((s, i) => (
                  <div key={i} style={{ background: C.mist, borderRadius: 10, padding: 14 }}>
                    <div style={{ fontFamily: C.font, fontSize: 11, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.04em', color: C.ink500 }}>{s.label}</div>
                    <div style={{ fontFamily: C.mono, fontSize: 18, fontWeight: 700, color: s.tone === 'warn' ? C.warn700 : s.tone === 'success' ? C.success700 : C.ink900, marginTop: 4 }}>
                      {s.value}
                    </div>
                  </div>
                ))}
              </div>
            );
          })()}

          <div style={{ marginBottom: 24 }}>
            <div style={{ fontFamily: C.font, fontSize: 16, fontWeight: 700, color: C.ink900, marginBottom: 8 }}>{t('grants_about') || 'Описание'}</div>
            <p style={{ fontFamily: C.font, fontSize: 14, lineHeight: '22px', color: C.ink700, margin: 0 }}>{grant.description}</p>
          </div>

          {grant.requirements && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ fontFamily: C.font, fontSize: 16, fontWeight: 700, color: C.ink900, marginBottom: 8 }}>Требования к кандидатам</div>
              <p style={{ fontFamily: C.font, fontSize: 14, lineHeight: '22px', color: C.ink700, margin: 0 }}>{grant.requirements}</p>
            </div>
          )}

          {grant.documents && (
            <div style={{ marginBottom: 24 }}>
              <div style={{ fontFamily: C.font, fontSize: 16, fontWeight: 700, color: C.ink900, marginBottom: 8 }}>Необходимые документы</div>
              <p style={{ fontFamily: C.font, fontSize: 14, lineHeight: '22px', color: C.ink700, margin: 0 }}>{grant.documents}</p>
            </div>
          )}

          <div style={{ borderRadius: 12, padding: 16, marginBottom: 28, background: C.blue50, border: `1px solid ${C.blue100}` }}>
            <div style={{ display: 'flex', gap: 10 }}>
              <div style={{ width: 28, height: 28, borderRadius: 9999, background: 'white', boxShadow: '0 0 0 3px rgba(20,72,255,0.06)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <Icon name="sparkles" size={14} color={C.blue} strokeWidth={2} />
              </div>
              <div>
                <div style={{ fontFamily: C.font, fontSize: 13, fontWeight: 700, color: C.ink900 }}>Нужна помощь в подготовке документов?</div>
                <div style={{ fontFamily: C.font, fontSize: 13, lineHeight: '20px', color: C.ink700, marginTop: 2 }}>
                  Наш AI-советник поможет написать мотивационное эссе, составить резюме и подготовиться к отбору на этот грант.
                </div>
              </div>
            </div>
          </div>

          <div style={{ display: 'flex', gap: 12 }}>
            <button
              onClick={askAI}
              style={{
                flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, padding: '14px 20px', borderRadius: 12,
                background: '#FFFFFF', border: `1px solid ${C.blue}`, cursor: 'pointer',
                fontFamily: C.font, fontSize: 15, fontWeight: 700, color: C.blue,
                transition: 'all 200ms'
              }}
              onMouseEnter={e => e.currentTarget.style.background = C.blue50}
              onMouseLeave={e => e.currentTarget.style.background = '#FFFFFF'}
            >
              <Icon name="sparkles" size={16} color={C.blue} />
              Спросить Talap AI
            </button>
            
            {grant.link && (
              <a
                href={grant.link}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, padding: '14px 20px', borderRadius: 12,
                  background: C.blue, border: 'none', cursor: 'pointer', textDecoration: 'none',
                  fontFamily: C.font, fontSize: 15, fontWeight: 700, color: '#FFFFFF',
                  transition: 'all 200ms', boxShadow: '0 4px 12px rgba(20,72,255,0.2)'
                }}
                onMouseEnter={e => e.currentTarget.style.transform = 'translateY(-2px)'}
                onMouseLeave={e => e.currentTarget.style.transform = 'translateY(0)'}
              >
                Официальный сайт
                <Icon name="arrowRight" size={16} color="#FFFFFF" />
              </a>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default function Grants() {
  const navigate = useNavigate();
  const { t, lang } = useLanguage();
  const isMobile = useMobile();
  const { aiProfile } = useAuth();
  const [grants, setGrants] = useState([]);
  const [countries, setCountries] = useState([]);
  const [portfolio, setPortfolio] = useState({});
  const [loading, setLoading] = useState(true);
  const [activeFilter, setActiveFilter] = useState('all');
  const [activeCountry, setActiveCountry] = useState('all');
  const [selectedGrant, setSelectedGrant] = useState(null);
  const [search, setSearch] = useState('');
  const isFirstRun = useRef(true);

  const FILTERS = [
    { key: 'all', label: t('cat_all') },
    { key: 'Бакалавриат', label: t('grants_bachelor') },
    { key: 'Магистратура', label: t('grants_master') },
    { key: 'IT', label: 'IT' },
    { key: 'Бизнес', label: t('cat_business') },
    { key: 'За рубежом', label: t('grants_abroad') },
    { key: 'Казахстан', label: t('grants_kz') },
  ];

  useEffect(() => {
    const query = new URLSearchParams(window.location.search).get('q');
    if (query) setSearch(query);

    Promise.all([
      api.get('/grants'),
      api.get('/portfolio')
    ])
      .then(([grantsData, portfolioData]) => {
        setGrants(grantsData);
        setCountries([...new Set(grantsData.map(g => g.country).filter(Boolean))]);
        setPortfolio(portfolioData);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (isFirstRun.current) {
      isFirstRun.current = false;
      return;
    }
    let cancelled = false;
    const params = new URLSearchParams();
    if (search) params.set('q', search);
    if (activeCountry !== 'all') params.set('country', activeCountry);
    if (activeFilter !== 'all') params.set('tags', activeFilter);
    const qs = params.toString();

    setLoading(true);
    api.get(`/grants${qs ? `?${qs}` : ''}`)
      .then(data => {
        if (cancelled) return;
        setGrants(data);
        setLoading(false);
      })
      .catch(() => { if (!cancelled) setLoading(false); });

    return () => { cancelled = true; };
  }, [activeFilter, activeCountry, search]);

  if (loading) return null;

  const filtered = [...grants].sort((a, b) => {
    const scoreA = calculateRealMatch(a, portfolio, false) || 0;
    const scoreB = calculateRealMatch(b, portfolio, false) || 0;
    return scoreB - scoreA;
  });

  return (
    <div style={{ flex: 1, overflow: 'auto', background: C.mist, display: 'flex', flexDirection: 'column' }}>
      <TopBar
        title={t('grants_title')}
        subtitle={`${grants.length} ${t('grants_subtitle')}`}
        actions={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Search value={search} onChange={setSearch} placeholder={t('grants_search')} />
          </div>
        }
      />

      <div style={{ padding: isMobile ? 16 : 32 }}>

        <div style={{ display: 'flex', gap: 8, marginBottom: 12, flexWrap: 'wrap' }}>
          {FILTERS.map(f => (
            <Chip key={f.key} selected={activeFilter === f.key} onClick={() => setActiveFilter(f.key)}>{f.label}</Chip>
          ))}
          <div style={{ flex: 1 }} />
          <Chip dot tone="ai" onClick={() => navigate('/chat')}>{t('grants_ai_match')}</Chip>
        </div>

        {countries.length > 0 && (
          <div style={{ display: 'flex', gap: 8, marginBottom: 24, flexWrap: 'wrap' }}>
            <Chip selected={activeCountry === 'all'} onClick={() => setActiveCountry('all')}>Все страны</Chip>
            {countries.map(c => (
              <Chip key={c} selected={activeCountry === c} onClick={() => setActiveCountry(c)}>{c}</Chip>
            ))}
          </div>
        )}

        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : '1fr 320px', gap: 24 }}>
          
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {filtered.length === 0 && (
              <Card>
                <div style={{ fontFamily: C.font, fontSize: 14, color: C.ink500, textAlign: 'center', padding: '24px 0' }}>
                  Ничего не найдено
                </div>
              </Card>
            )}
            {filtered.map(g => (
              <Card key={g.id} onClick={() => setSelectedGrant(g)} style={{ cursor: 'pointer' }}>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: 16, alignItems: 'flex-start' }}>
                  <div>
                    <div style={{ display: 'flex', gap: 6, marginBottom: 10, flexWrap: 'wrap' }}>
                      {(g.tags || ['Грант']).map(tag => <Chip key={tag}>{tag}</Chip>)}
                    </div>
                    <div style={{ fontFamily: C.font, fontSize: 17, fontWeight: 700, letterSpacing: '-0.015em', color: C.ink900, marginBottom: 4 }}>{g.name}</div>
                    <div style={{ fontFamily: C.font, fontSize: 13, color: C.ink500 }}>{g.subtitle} · {g.country}</div>
                  </div>
                  <div style={{ textAlign: 'right', flexShrink: 0 }}>
                    {(() => {
                      const score = calculateRealMatch(g, portfolio, false);
                      return score ? (
                        <div style={{ fontFamily: C.mono, fontSize: 13, fontWeight: 700, color: C.blue, marginBottom: 4 }}>{score}% match</div>
                      ) : null;
                    })()}
                    <div style={{ fontFamily: C.font, fontSize: 12, color: g.tone === 'warn' ? C.warn700 : C.ink500, marginBottom: 8 }}>
                      {g.deadline_days} {t('grants_deadline')}
                    </div>
                  </div>
                </div>
                {(g.source_url || g.fetched_at) && (
                  <div style={{ fontFamily: C.font, fontSize: 11, color: C.ink300, marginTop: 10, paddingTop: 10, borderTop: `1px solid ${C.hairline}`, display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                    {g.source_url && (
                      <span onClick={e => e.stopPropagation()}>
                        {t('grant_source_label')}: <a href={g.source_url} target="_blank" rel="noreferrer" style={{ color: C.ink300 }}>{g.source_url}</a>
                      </span>
                    )}
                    {g.fetched_at && (
                      <span>{t('grant_updated_label')}: {new Date(g.fetched_at).toLocaleDateString(lang === 'kz' ? 'kk-KZ' : lang === 'en' ? 'en-US' : 'ru-RU')}</span>
                    )}
                  </div>
                )}
              </Card>
            ))}
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <Card>
              <div style={{ fontFamily: C.font, fontSize: 13, fontWeight: 700, color: C.ink900, marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
                <Icon name="target" size={16} color={C.blue} />
                {t('grants_match_profile')}
              </div>
              {[
                { label: 'ЕНТ', value: portfolio.ent || '-', bar: portfolio.ent ? (parseInt(portfolio.ent)/140)*100 : 0 },
                { label: 'English', value: portfolio.english || '-', bar: portfolio.english ? 80 : 0 },
              ].map((s, i) => (
                <div key={i} style={{ marginBottom: 14 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                    <span style={{ fontFamily: C.font, fontSize: 12, color: C.ink500 }}>{s.label}</span>
                    <span style={{ fontFamily: C.mono, fontSize: 12, fontWeight: 600, color: C.ink900 }}>{s.value}</span>
                  </div>
                  <Progress value={s.bar} height={4} />
                </div>
              ))}
            </Card>
          </div>
        </div>
      </div>

      {selectedGrant && <GrantDetailModal grant={selectedGrant} t={t} lang={lang} portfolio={portfolio} onClose={() => setSelectedGrant(null)} />}
    </div>
  );
}
