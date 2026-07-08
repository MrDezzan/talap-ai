import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import Icon from '../components/Icon';
import Button from '../components/Button';
import Chip from '../components/Chip';
import { useLanguage } from '../context/LanguageContext';
import { useMobile } from '../hooks/useMobile';
import avatar1 from '../assets/avatar1.png';
import avatar2 from '../assets/avatar2.png';
import avatar3 from '../assets/avatar3.png';
import avatar4 from '../assets/avatar4.png';

const C = {
  ink900: '#0A1230', ink700: '#2A3457', ink500: '#5A6485', ink300: '#9AA3BF',
  blue: '#1448FF', blue50: '#F2F5FF', blue100: '#E6ECFF',
  paper: '#FFFFFF', mist: '#F5F7FB', mist2: '#ECEFF7', hairline: '#E4E8F1',
  success700: '#047857', success100: '#ECFAF3',
  font: 'var(--font-sans)',
};

function LanguageSwitcher() {
  const { lang, setLang } = useLanguage();
  return (
    <div style={{ display: 'flex', gap: 4, background: C.mist, padding: 4, borderRadius: 8 }}>
      {['ru', 'en', 'kz'].map(l => (
        <button
          key={l}
          onClick={() => setLang(l)}
          style={{
            padding: '4px 8px', borderRadius: 6, border: 'none',
            background: lang === l ? 'white' : 'transparent',
            color: lang === l ? C.blue : C.ink500,
            fontSize: 11, fontWeight: 700, textTransform: 'uppercase',
            cursor: 'pointer', boxShadow: lang === l ? '0 2px 4px rgba(10,18,48,0.05)' : 'none',
            transition: 'all 200ms'
          }}
        >
          {l}
        </button>
      ))}
    </div>
  );
}

function Navbar() {
  const { t } = useLanguage();
  const isMobile = useMobile();
  return (
    <header style={{
      position: 'sticky', top: 0, zIndex: 100,
      background: 'rgba(6,13,31,0.85)', backdropFilter: 'blur(20px)',
      borderBottom: `1px solid rgba(0,212,255,0.12)`,
    }}>
      <div style={{ maxWidth: 1200, margin: '0 auto', padding: isMobile ? '0 16px' : '0 40px', height: 64, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Link to="/" style={{ display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', flexShrink: 0 }}>
          <span style={{ fontFamily: C.font, fontSize: 18, fontWeight: 800, letterSpacing: '-0.03em', color: '#FFFFFF' }}>
            Talap<span style={{ color: '#00D4FF' }}>.</span>ai
          </span>
        </Link>

        {!isMobile && (
          <nav style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            {[
              { label: t('land_nav_features'), href: '#Возможности' },
              { label: t('land_nav_how'), href: '#how' },
              { label: t('land_nav_grants'), href: '/grants' },
            ].map(item => (
              item.href.startsWith('/') ? (
                <Link key={item.label} to={item.href} style={{
                  fontFamily: C.font, fontSize: 14, fontWeight: 500, color: 'rgba(255,255,255,0.7)',
                  padding: '6px 12px', borderRadius: 6, textDecoration: 'none',
                  transition: 'color 150ms',
                }}>{item.label}</Link>
              ) : (
                <a key={item.label} href={item.href} style={{
                  fontFamily: C.font, fontSize: 14, fontWeight: 500, color: 'rgba(255,255,255,0.7)',
                  padding: '6px 12px', borderRadius: 6, textDecoration: 'none',
                  transition: 'color 150ms',
                }}>{item.label}</a>
              )
            ))}
          </nav>
        )}

        <div style={{ display: 'flex', alignItems: 'center', gap: isMobile ? 8 : 16 }}>
          <LanguageSwitcher />
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            {!isMobile && (
              <Link to="/login">
                <Button variant="ghost" size="sm" style={{ color: 'rgba(255,255,255,0.7)' }}>{t('land_nav_login')}</Button>
              </Link>
            )}
            <Link to="/register">
              <Button variant="primary" size="sm">{t('land_nav_start')}</Button>
            </Link>
          </div>
        </div>
      </div>
    </header>
  );
}

function Hero() {
  const navigate = useNavigate();
  const { t, lang } = useLanguage();
  const isMobile = useMobile();
  const [wordIndex, setWordIndex] = React.useState(0);
  const [displayed, setDisplayed] = React.useState('');
  const [deleting, setDeleting] = React.useState(false);
  const [activeCard, setActiveCard] = React.useState(0);

  const words = lang === 'kz'
    ? ['БАҒДАРЛАМАШЫ', 'ДӘРІГЕР', 'БИЗНЕСМЕН', 'ДИЗАЙНЕР', 'AI-МАМАН']
    : lang === 'en'
    ? ['PROGRAMMER', 'DOCTOR', 'ENTREPRENEUR', 'DESIGNER', 'AI SPECIALIST']
    : ['ПРОГРАММИСТА', 'ВРАЧА', 'БИЗНЕСМЕНА', 'ДИЗАЙНЕРА', 'AI-СПЕЦИАЛИСТА'];

  React.useEffect(() => {
    const word = words[wordIndex];
    let timeout;
    if (!deleting && displayed.length < word.length) {
      timeout = setTimeout(() => setDisplayed(word.slice(0, displayed.length + 1)), 80);
    } else if (!deleting && displayed.length === word.length) {
      timeout = setTimeout(() => setDeleting(true), 1800);
    } else if (deleting && displayed.length > 0) {
      timeout = setTimeout(() => setDisplayed(displayed.slice(0, -1)), 45);
    } else if (deleting && displayed.length === 0) {
      setDeleting(false);
      setWordIndex((wordIndex + 1) % words.length);
    }
    return () => clearTimeout(timeout);
  }, [displayed, deleting, wordIndex, words]);

  // Auto-rotate cards
  React.useEffect(() => {
    const t = setInterval(() => setActiveCard(p => (p + 1) % 3), 3200);
    return () => clearInterval(t);
  }, []);

  const cards = [
    {
      label: lang === 'kz' ? 'БАҒДАРЛАМАШЫ' : lang === 'en' ? 'PROGRAMMER' : 'ПРОГРАММИСТ',
      color: '#00D4FF',
      glow: 'rgba(0,212,255,0.45)',
      border: 'rgba(0,212,255,0.5)',
      bg: 'rgba(0,212,255,0.07)',
      icon: '💻',
      timeline: lang === 'ru' ? ['Основы', 'Алгоритмы', 'Фреймворки', 'Оффер'] : ['Basics', 'Algorithms', 'Frameworks', 'Offer'],
    },
    {
      label: lang === 'kz' ? 'ДӘРІГЕР' : lang === 'en' ? 'DOCTOR' : 'ВРАЧ',
      color: '#00E5A0',
      glow: 'rgba(0,229,160,0.4)',
      border: 'rgba(0,229,160,0.5)',
      bg: 'rgba(0,229,160,0.07)',
      icon: '🩺',
      timeline: lang === 'ru' ? ['Биология', 'Медвуз', 'Интернатура', 'Специализация'] : ['Biology', 'Medical', 'Internship', 'Speciality'],
    },
    {
      label: lang === 'kz' ? 'БИЗНЕСМЕН' : lang === 'en' ? 'ENTREPRENEUR' : 'БИЗНЕСМЕН',
      color: '#FF8C42',
      glow: 'rgba(255,140,66,0.4)',
      border: 'rgba(255,140,66,0.5)',
      bg: 'rgba(255,140,66,0.07)',
      icon: '📊',
      timeline: lang === 'ru' ? ['Идея', 'MVP', 'Инвестиции', 'Масштаб'] : ['Idea', 'MVP', 'Investment', 'Scale'],
    },
  ];

  return (
    <div style={{ position: 'relative', overflow: 'hidden', background: '#060D1F' }}>
      {/* Background image */}
      <div style={{
        position: 'absolute', inset: 0,
        backgroundImage: 'url(/hero_bg.png)',
        backgroundSize: 'cover',
        backgroundPosition: 'center top',
        opacity: 0.18,
        zIndex: 0,
      }} />
      {/* Animated grid */}
      <div style={{
        position: 'absolute', inset: 0, zIndex: 0,
        backgroundImage: `
          linear-gradient(rgba(0,212,255,0.04) 1px, transparent 1px),
          linear-gradient(90deg, rgba(0,212,255,0.04) 1px, transparent 1px)
        `,
        backgroundSize: '48px 48px',
      }} />
      {/* Top glow */}
      <div style={{
        position: 'absolute', top: -120, left: '50%', transform: 'translateX(-50%)',
        width: 800, height: 400,
        background: 'radial-gradient(ellipse, rgba(20,72,255,0.25) 0%, transparent 70%)',
        zIndex: 0,
      }} />

      <style>{`
        @keyframes blink { 0%,100%{opacity:1} 50%{opacity:0} }
        @keyframes cardFloat {
          0%,100%{transform:translateY(0px)}
          50%{transform:translateY(-10px)}
        }
        @keyframes glowPulse {
          0%,100%{box-shadow:0 0 30px var(--glow-color), 0 0 60px var(--glow-color), inset 0 0 20px rgba(255,255,255,0.03)}
          50%{box-shadow:0 0 50px var(--glow-color), 0 0 100px var(--glow-color), inset 0 0 30px rgba(255,255,255,0.06)}
        }
        @keyframes fadeUp {
          from{opacity:0;transform:translateY(30px)}
          to{opacity:1;transform:translateY(0)}
        }
        .hero-card { animation: cardFloat 4s ease-in-out infinite; }
        .hero-card:nth-child(2) { animation-delay: -1.2s; }
        .hero-card:nth-child(3) { animation-delay: -2.5s; }
        .hero-cta:hover { transform: translateY(-3px) scale(1.02) !important; box-shadow: 0 0 40px rgba(20,72,255,0.6), 0 16px 40px rgba(20,72,255,0.4) !important; }
        .hero-cta:active { transform: scale(0.98) !important; }
      `}</style>

      <section style={{
        padding: isMobile ? '60px 16px 52px' : '90px 40px 80px',
        maxWidth: 1200, margin: '0 auto', position: 'relative', zIndex: 1,
        animation: 'fadeUp 0.7s ease forwards',
      }}>
        {/* Stars / badges */}
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: 28 }}>
          <div style={{
            display: 'inline-flex', alignItems: 'center', gap: 8,
            background: 'rgba(0,212,255,0.1)', border: '1px solid rgba(0,212,255,0.3)',
            borderRadius: 24, padding: '6px 16px',
          }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#00D4FF', boxShadow: '0 0 8px #00D4FF', display: 'inline-block' }} />
            <span style={{ fontFamily: C.font, fontSize: 12, fontWeight: 700, color: 'rgba(0,212,255,0.9)', letterSpacing: '0.08em', textTransform: 'uppercase' }}>
              {lang === 'kz' ? 'AI-мансап жоспарлаушы' : lang === 'en' ? 'AI Career Planner' : 'AI Карьерный советник · Казахстан'}
            </span>
          </div>
        </div>

        {/* Headline */}
        <h1 style={{
          fontFamily: C.font,
          fontSize: isMobile ? 32 : 58,
          lineHeight: 1.1,
          fontWeight: 900,
          letterSpacing: '-0.03em',
          color: '#FFFFFF',
          textAlign: 'center',
          margin: '0 0 20px',
        }}>
          {lang === 'kz' ? 'МАНСАПҚА' : lang === 'en' ? 'YOUR IDEAL PATH' : 'ТВОЙ ИДЕАЛЬНЫЙ ПУТЬ'}<br />
          {lang === 'kz' ? 'НАҚТЫ ЖОЛ' : lang === 'en' ? 'TO CAREER OF' : 'К КАРЬЕРЕ'}{' '}
          <span style={{ color: '#00D4FF', textShadow: '0 0 30px rgba(0,212,255,0.8)' }}>
            [{displayed}
            <span style={{ animation: 'blink 1s step-end infinite', color: '#00D4FF' }}>|</span>]
          </span>
        </h1>

        {/* Sub-headline */}
        <p style={{
          fontFamily: C.font,
          fontSize: isMobile ? 15 : 18,
          lineHeight: 1.6,
          color: 'rgba(255,255,255,0.6)',
          textAlign: 'center',
          margin: '0 auto 40px',
          maxWidth: 560,
        }}>
          {lang === 'kz'
            ? 'Мамандық таңдаудан бастап бірінші офферге дейін — дамудың қадамдық жоспарын ал.'
            : lang === 'en'
            ? 'From choosing a specialty to your first job offer — get a step-by-step development plan.'
            : 'От выбора специальности до первого оффера — получи пошаговый план развития.'}
        </p>

        {/* CTA */}
        <div style={{ display: 'flex', justifyContent: 'center', marginBottom: isMobile ? 52 : 72 }}>
          <button
            className="hero-cta"
            onClick={() => navigate('/register')}
            style={{
              display: 'inline-flex', alignItems: 'center', gap: 12,
              padding: isMobile ? '16px 32px' : '18px 44px',
              borderRadius: 14,
              background: 'linear-gradient(135deg, #1448FF 0%, #0A30CC 100%)',
              border: '1px solid rgba(0,212,255,0.4)',
              fontFamily: C.font, fontSize: isMobile ? 15 : 17, fontWeight: 800,
              color: '#FFFFFF', cursor: 'pointer', letterSpacing: '0.03em',
              boxShadow: '0 0 24px rgba(20,72,255,0.4), 0 8px 24px rgba(20,72,255,0.3)',
              transition: 'all 250ms cubic-bezier(0.34,1.56,0.64,1)',
              textTransform: 'uppercase',
            }}
          >
            <span>⚡</span>
            {lang === 'kz' ? 'МЕНІҢ ЖОЛЫМДЫ БАСТАУ' : lang === 'en' ? 'START MY PATH' : 'НАЧАТЬ МОЙ ПУТЬ'}
          </button>
        </div>

        {/* Glassmorphism career cards */}
        <div style={{
          display: 'flex', gap: isMobile ? 12 : 24,
          justifyContent: 'center',
          flexWrap: isMobile ? 'wrap' : 'nowrap',
        }}>
          {cards.map((card, i) => {
            const isActive = activeCard === i;
            return (
              <div
                key={i}
                className="hero-card"
                onClick={() => setActiveCard(i)}
                style={{
                  '--glow-color': card.glow,
                  flex: isMobile ? '0 0 calc(50% - 6px)' : '0 0 220px',
                  maxWidth: isMobile ? 180 : 220,
                  borderRadius: 20,
                  padding: '20px 16px 18px',
                  background: isActive
                    ? `linear-gradient(160deg, ${card.bg}, rgba(0,0,0,0.3))`
                    : 'rgba(255,255,255,0.03)',
                  border: `1px solid ${isActive ? card.border : 'rgba(255,255,255,0.08)'}`,
                  backdropFilter: 'blur(16px)',
                  cursor: 'pointer',
                  transition: 'all 400ms cubic-bezier(0.34,1.2,0.64,1)',
                  boxShadow: isActive
                    ? `0 0 32px ${card.glow}, 0 0 64px ${card.glow}`
                    : '0 4px 24px rgba(0,0,0,0.2)',
                  transform: isActive ? 'translateY(-8px) scale(1.04)' : 'scale(1)',
                  animation: `cardFloat ${3.8 + i * 0.4}s ease-in-out infinite`,
                  animationDelay: `${-i * 1.2}s`,
                }}
              >
                {/* Icon */}
                <div style={{
                  fontSize: 32, marginBottom: 12, textAlign: 'center',
                  filter: isActive ? `drop-shadow(0 0 12px ${card.color})` : 'none',
                  transition: 'filter 400ms',
                }}>
                  {card.icon}
                </div>

                {/* Label */}
                <div style={{
                  fontFamily: C.font, fontSize: 13, fontWeight: 800,
                  color: isActive ? card.color : 'rgba(255,255,255,0.5)',
                  textAlign: 'center', letterSpacing: '0.06em',
                  marginBottom: 16, transition: 'color 300ms',
                  textShadow: isActive ? `0 0 16px ${card.color}` : 'none',
                }}>
                  {card.label}
                </div>

                {/* Timeline dots */}
                <div style={{ position: 'relative' }}>
                  {/* Line */}
                  <div style={{
                    position: 'absolute', top: 6, left: 8, right: 8,
                    height: 1, background: isActive
                      ? `linear-gradient(90deg, transparent, ${card.color}, transparent)`
                      : 'rgba(255,255,255,0.1)',
                    transition: 'background 400ms',
                  }} />
                  <div style={{ display: 'flex', justifyContent: 'space-between', position: 'relative' }}>
                    {card.timeline.map((step, j) => (
                      <div key={j} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, flex: 1 }}>
                        <div style={{
                          width: 12, height: 12, borderRadius: '50%',
                          background: isActive ? card.color : 'rgba(255,255,255,0.15)',
                          boxShadow: isActive ? `0 0 10px ${card.color}` : 'none',
                          border: `2px solid ${isActive ? card.color : 'rgba(255,255,255,0.2)'}`,
                          transition: 'all 400ms',
                          transitionDelay: `${j * 60}ms`,
                        }} />
                        <span style={{
                          fontFamily: C.font, fontSize: 8, fontWeight: 600,
                          color: isActive ? 'rgba(255,255,255,0.7)' : 'rgba(255,255,255,0.25)',
                          textAlign: 'center', lineHeight: 1.2, transition: 'color 400ms',
                          maxWidth: 40,
                        }}>
                          {step}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {/* Stats row */}
        <div style={{
          display: 'flex', gap: isMobile ? 24 : 56, justifyContent: 'center',
          marginTop: isMobile ? 44 : 64,
          borderTop: '1px solid rgba(255,255,255,0.07)',
          paddingTop: isMobile ? 32 : 44,
          flexWrap: 'wrap',
        }}>
          {[
            { value: '50 000+', label: lang === 'kz' ? 'белсенді оқушы' : lang === 'en' ? 'active students' : 'активных студентов' },
            { value: '1 200+', label: lang === 'kz' ? 'грант базасы' : lang === 'en' ? 'grants in base' : 'грантов в базе' },
            { value: '95%', label: lang === 'kz' ? 'сәтті нәтиже' : lang === 'en' ? 'success rate' : 'успешных результатов' },
            { value: '13', label: lang === 'kz' ? 'мамандық' : lang === 'en' ? 'career paths' : 'специальностей' },
          ].map(s => (
            <div key={s.label} style={{ textAlign: 'center' }}>
              <div style={{
                fontFamily: C.font, fontSize: isMobile ? 26 : 34, fontWeight: 900,
                letterSpacing: '-0.03em',
                background: 'linear-gradient(135deg, #FFFFFF 0%, rgba(0,212,255,0.9) 100%)',
                WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
              }}>{s.value}</div>
              <div style={{ fontFamily: C.font, fontSize: 12, color: 'rgba(255,255,255,0.4)', marginTop: 4 }}>{s.label}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

function Features() {
  const { t, lang } = useLanguage();
  const isMobile = useMobile();
  const features = [
    {
      icon: 'compass',
      title: lang === 'en' ? 'Proforientation' : lang === 'kz' ? 'Мамандық таңдау' : 'Профориентация',
      desc: lang === 'en' ? 'Take a 24-question test — get a list of professions with match percentages.' : 
            lang === 'kz' ? '24 сұрақтан тұратын тест тапсырып, мамандықтар тізімін алыңыз.' :
            'Пройди тест из 24 вопросов — получи список профессий с процентом совпадения с твоим профилем.',
      color: C.blue, bg: C.blue100,
    },
    {
      icon: 'award',
      title: t('nav_grants'),
      desc: lang === 'en' ? '1,200+ current grants in Kazakhstan and abroad. AI picks based on your grades.' :
            lang === 'kz' ? 'Қазақстандағы және шетелдегі 1 200-ден астам гранттар. AI сенің бағаларыңа қарай таңдайды.' :
            '1 200+ актуальных грантов в Казахстане и за рубежом. AI подбирает по твоим оценкам и направлению.',
      color: '#047857', bg: '#ECFAF3',
    },
    {
      icon: 'target',
      title: t('nav_roadmap'),
      desc: lang === 'en' ? 'Step-by-step roadmap from school to the first job. With specific courses and deadlines.' :
            lang === 'kz' ? 'Мектептен алғашқы жұмысқа дейінгі қадамдық жол картасы. Нақты курстар мен дедлайндармен.' :
            'Пошаговый roadmap от школы до первой работы. С конкретными курсами, дедлайнами и чекпоинтами.',
      color: '#6366F1', bg: '#EEF0FF',
    },
    {
      icon: 'trophy',
      title: t('nav_portfolio'),
      desc: lang === 'en' ? 'Collect Olympiads, courses, and hackathons in one place. Export for grant applications.' :
            lang === 'kz' ? 'Олимпиадаларды, курстарды, хакатондарды бір жерге жинаңыз. Грантқа өтінім үшін экспорттаңыз.' :
            'Собери олимпиады, курсы, хакатоны в одном месте. Экспортируй для заявки на грант одним кликом.',
      color: '#8A5A0E', bg: '#FFF6E5',
    },
    {
      icon: 'sparkles',
      title: t('chat_title'),
      desc: lang === 'en' ? 'Ask anything about careers, universities, or grants. Remembers your profile.' :
            lang === 'kz' ? 'Мансап, ЖОО немесе гранттар туралы кез келген нәрсені сұраңыз. Сенің профиліңді есте сақтайды.' :
            'Спроси что угодно о карьере, вузах или грантах. Отвечает на русском и казахском, помнит твой профиль.',
      color: C.blue, bg: C.blue50,
    },
    {
      icon: 'bell',
      title: lang === 'en' ? 'Deadlines & Reminders' : lang === 'kz' ? 'Дедлайндар мен еске салғыштар' : 'Дедлайнды и напоминания',
      desc: lang === 'en' ? 'Never miss a deadline. Smart notifications 30, 14, and 7 days before.' :
            lang === 'kz' ? 'Маңызды мерзімді ешқашан жіберіп алмаңыз. 30, 14 және 7 күн бұрын ақылды хабарламалар.' :
            'Никогда не пропустишь важный срок. Умные уведомления за 30, 14 и 7 дней до дедлайна.',
      color: '#B42318', bg: '#FDECEC',
    },
  ];

  return (
    <section id="Возможности" style={{ padding: isMobile ? '64px 16px' : '80px 40px', background: C.mist }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ textAlign: 'center', marginBottom: 56 }}>
          <Chip>{t('land_nav_features')}</Chip>
          <h2 style={{ fontFamily: C.font, fontSize: isMobile ? 32 : 40, fontWeight: 800, letterSpacing: '-0.03em', color: C.ink900, margin: '16px 0 16px' }}>
            {t('land_features_title')}
          </h2>
          <p style={{ fontFamily: C.font, fontSize: isMobile ? 15 : 17, color: C.ink700, maxWidth: 520, margin: '0 auto' }}>
            {t('land_features_desc')}
          </p>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : 'repeat(3, 1fr)', gap: 20 }}>
          {features.map((f, i) => (
            <div key={i} style={{ background: C.paper, borderRadius: 16, padding: isMobile ? 24 : 28, border: `1px solid ${C.hairline}`, transition: 'transform 200ms, box-shadow 200ms' }}>
              <div style={{ width: 48, height: 48, borderRadius: 12, background: f.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 20 }}>
                <Icon name={f.icon} size={24} color={f.color} strokeWidth={1.75} />
              </div>
              <div style={{ fontFamily: C.font, fontSize: 18, fontWeight: 700, letterSpacing: '-0.02em', color: C.ink900, marginBottom: 10 }}>{f.title}</div>
              <div style={{ fontFamily: C.font, fontSize: 14, lineHeight: '22px', color: C.ink500 }}>{f.desc}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function HowItWorks() {
  const { t, lang } = useLanguage();
  const isMobile = useMobile();
  const steps = [
    { n: '01', title: lang === 'en' ? 'Tell about yourself' : lang === 'kz' ? 'Өзің туралы айт' : 'Расскажи о себе', desc: lang === 'en' ? 'Fill out your profile: grade, grades, interests.' : lang === 'kz' ? 'Профильді толтыр: сынып, бағалар, қызығушылықтар.' : 'Заполни профиль: класс, оценки, интересы.' },
    { n: '02', title: lang === 'en' ? 'Take the test' : lang === 'kz' ? 'Тест тапсыр' : 'Пройди тест', desc: lang === 'en' ? '24 questions and Talap AI picks professions.' : lang === 'kz' ? '24 сұрақ — Talap AI мамандық таңдайды.' : '24 вопроса — и Talap AI подберёт профессии.' },
    { n: '03', title: lang === 'en' ? 'Get a plan' : lang === 'kz' ? 'Жоспар ал' : 'Получи план', desc: lang === 'en' ? 'Personal roadmap with courses and grants.' : lang === 'kz' ? 'Курстар мен гранттары бар жол картасы.' : 'Персональный roadmap с курсами, грантами и дедлайнами.' },
    { n: '04', title: lang === 'en' ? 'Take action' : lang === 'kz' ? 'Әрекет ет' : 'Действуй', desc: lang === 'en' ? 'Apply, collect portfolio, ask AI.' : lang === 'kz' ? 'Өтінім бер, портфолио жина, AI-дан сұра.' : 'Подавай заявки, собирай портфолио, задавай вопросы AI.' },
  ];
  return (
    <section id="how" style={{ padding: isMobile ? '64px 16px' : '80px 40px' }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ textAlign: 'center', marginBottom: 56 }}>
          <Chip>{t('land_nav_how')}</Chip>
          <h2 style={{ fontFamily: C.font, fontSize: isMobile ? 32 : 40, fontWeight: 800, letterSpacing: '-0.03em', color: C.ink900, margin: '16px 0 16px' }}>
            {lang === 'en' ? 'Four steps to your dream career' : lang === 'kz' ? 'Арман мансабына төрт қадам' : 'Четыре шага до карьеры мечты'}
          </h2>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : 'repeat(4, 1fr)', gap: isMobile ? 32 : 0, position: 'relative' }}>
          {!isMobile && <div style={{ position: 'absolute', top: 28, left: '12.5%', right: '12.5%', height: 1, background: `linear-gradient(90deg, ${C.blue}, ${C.blue}66)` }} />}
          {steps.map((s, i) => (
            <div key={i} style={{ padding: isMobile ? 0 : '0 20px', textAlign: 'center' }}>
              <div style={{
                width: 56, height: 56, borderRadius: 9999, background: i === 0 ? C.blue : 'white',
                border: `2px solid ${i === 0 ? C.blue : C.hairline}`,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                margin: '0 auto 20px',
                fontFamily: '"Geist Mono",monospace', fontSize: 15, fontWeight: 700,
                color: i === 0 ? 'white' : C.ink500,
                position: 'relative', zIndex: 1,
              }}>{s.n}</div>
              <div style={{ fontFamily: C.font, fontSize: 17, fontWeight: 700, color: C.ink900, marginBottom: 10 }}>{s.title}</div>
              <div style={{ fontFamily: C.font, fontSize: 14, lineHeight: '22px', color: C.ink500 }}>{s.desc}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function Testimonials() {
  const { lang } = useLanguage();
  const isMobile = useMobile();
  const testimonials = [
    {
      text: lang === 'en' ? 'Thanks to Talap found Bolashak and entered UK master program.' : lang === 'kz' ? 'Talap арқасында Болашақ грантын тауып, Ұлыбританияға магистратураға түстім.' : 'Благодаря Talap нашла грант Болашак и поступила в магистратуру в Великобритании.',
      name: 'Айгерим Нурланова', role: 'Student, University of Edinburgh', tone: 'blue',
    },
    {
      text: lang === 'en' ? 'The test showed Data Science. Now intern at Kaspi.kz.' : lang === 'kz' ? 'Тест Data Science көрсетті. Қазір Kaspi.kz-те стажермын.' : 'Тест профориентации показал Data Science — сначала не верил. Сейчас стажёр в Kaspi.kz.',
      name: 'Даниял Сейткали', role: 'NU Student, Kaspi.kz Intern', tone: 'iris',
    },
    {
      text: lang === 'en' ? 'Career roadmap put everything in its place. Invaluable.' : lang === 'kz' ? 'Мансап жол картасы бәрін реттеді. Өте құнды.' : 'Карьерный roadmap разложил всё по полочкам. Я знал что сдавать, где учиться и как строить портфолио. Это бесценно.',
      name: 'Алибек Джаксыбеков', role: 'Developer at Chocofamily', tone: 'amber',
    },
  ];
  return (
    <section style={{ padding: isMobile ? '64px 16px' : '80px 40px', background: C.mist }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ textAlign: 'center', marginBottom: 56 }}>
          <Chip tone="success" dot>{lang === 'en' ? 'Success Stories' : lang === 'kz' ? 'Жетістік тарихы' : 'Истории успеха'}</Chip>
          <h2 style={{ fontFamily: C.font, fontSize: isMobile ? 32 : 40, fontWeight: 800, letterSpacing: '-0.03em', color: C.ink900, margin: '16px 0 0' }}>
            {lang === 'en' ? 'They already found their way' : lang === 'kz' ? 'Олар өз жолын тапты' : 'Они уже нашли свой путь'}
          </h2>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : 'repeat(3, 1fr)', gap: 20 }}>
          {testimonials.map((t, i) => (
            <div key={i} style={{ background: 'white', borderRadius: 16, padding: isMobile ? 24 : 28, border: `1px solid ${C.hairline}` }}>
              <div style={{ display: 'flex', gap: 2, marginBottom: 16 }}>
                {[...Array(5)].map((_, j) => <Icon key={j} name="star" size={16} color="#F2C572" style={{ fill: '#F2C572' }} />)}
              </div>
              <p style={{ fontFamily: C.font, fontSize: 15, lineHeight: '24px', color: C.ink700, margin: '0 0 20px' }}>«{t.text}»</p>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={{ width: 40, height: 40, borderRadius: 9999, background: C.blue50, color: C.blue, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: C.font, fontWeight: 700, fontSize: 16 }}>{t.name[0]}</div>
                <div>
                  <div style={{ fontFamily: C.font, fontSize: 14, fontWeight: 700, color: C.ink900 }}>{t.name}</div>
                  <div style={{ fontFamily: C.font, fontSize: 12, color: C.ink500 }}>{t.role}</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function CTABanner() {
  const { t } = useLanguage();
  const isMobile = useMobile();
  return (
    <section style={{ padding: isMobile ? '48px 16px' : '80px 40px' }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ borderRadius: 24, padding: isMobile ? '48px 24px' : '64px 80px', textAlign: 'center', background: C.blue50 }}>
          <h2 style={{ fontFamily: C.font, fontSize: isMobile ? 32 : 44, fontWeight: 800, letterSpacing: '-0.03em', color: C.ink900, margin: '0 0 16px' }}>
            {t('land_cta_title')}
          </h2>
          <p style={{ fontFamily: C.font, fontSize: isMobile ? 15 : 17, color: C.ink700, margin: '0 0 36px', maxWidth: 520, marginLeft: 'auto', marginRight: 'auto' }}>
            {t('land_cta_desc')}
          </p>
          <div style={{ display: 'flex', gap: 12, justifyContent: 'center', flexDirection: isMobile ? 'column' : 'row' }}>
            <Link to="/register"><Button variant="primary" size="lg" icon="arrowRight" fullWidth={isMobile}>{t('land_cta_btn_profile')}</Button></Link>
            <Link to="/login"><Button variant="outline" size="lg" fullWidth={isMobile}>{t('land_cta_btn_login')}</Button></Link>
          </div>
        </div>
      </div>
    </section>
  );
}

function Footer() {
  const { lang } = useLanguage();
  const isMobile = useMobile();
  return (
    <footer style={{ background: C.ink900, padding: isMobile ? '40px 16px' : '56px 40px 40px' }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        <div style={{ borderTop: '1px solid #2A3457', paddingTop: 24, display: 'flex', justifyContent: 'space-between', alignItems: isMobile ? 'flex-start' : 'center', flexDirection: isMobile ? 'column' : 'row', gap: 16 }}>
          <div style={{ fontFamily: C.font, fontSize: 13, color: '#5A6485' }}>© 2026 Talap.ai. {lang === 'en' ? 'All rights reserved.' : lang === 'kz' ? 'Барлық құқықтар қорғалған.' : 'Все права защищены.'}</div>
          <div style={{ fontFamily: C.font, fontSize: 13, color: '#5A6485' }}>{lang === 'en' ? 'Astana, Kazakhstan' : 'Астана, Казахстан'} 🇰🇿</div>
        </div>
      </div>
    </footer>
  );
}

export default function Landing() {
  return (
    <div style={{ background: '#FFFFFF', overflowX: 'hidden' }}>
      <Navbar />
      <Hero />
      <Features />
      <HowItWorks />
      <Testimonials />
      <CTABanner />
      <Footer />
    </div>
  );
}
