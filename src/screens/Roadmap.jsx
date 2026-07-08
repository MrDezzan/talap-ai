import { useState, useEffect } from 'react';
import { useMobile } from '../hooks/useMobile';
import { useNavigate } from 'react-router-dom';
import Icon from '../components/Icon';
import TopBar from '../components/TopBar';
import Card from '../components/Card';
import Button from '../components/Button';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';

const C = {
  ink900: '#0A1230', ink700: '#2A3457', ink500: '#5A6485',
  blue: '#1448FF', blue50: '#F2F5FF', blue100: '#E6ECFF',
  paper: '#FFFFFF', mist: '#F5F7FB', hairline: '#E4E8F1',
  font: 'var(--font-sans)',
};

export default function Roadmap() {
  const navigate = useNavigate();
  const { aiProfile } = useAuth();
  const { t, lang } = useLanguage();
  const steps = aiProfile?.roadmap || [];
  const isMobile = useMobile();

  return (
    <div style={{ flex: 1, overflow: 'auto', background: C.mist, display: 'flex', flexDirection: 'column' }}>
      <TopBar title={t('nav_roadmap')} />

      <div style={{ padding: isMobile ? '16px' : '40px', maxWidth: 800, margin: '0 auto', width: '100%', boxSizing: 'border-box' }}>
        {steps.length > 0 ? (
          <>
            <div style={{ marginBottom: 40, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: 16 }} className="no-print">
              <div>
                <div style={{ fontFamily: C.font, fontSize: 14, fontWeight: 700, color: C.blue, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 8 }}>
                  {t('dash_roadmap_title')}
                </div>
                <h1 style={{ fontFamily: C.font, fontSize: 32, fontWeight: 800, color: C.ink900, letterSpacing: '-0.03em', margin: 0 }}>
                  {aiProfile.top_profession}
                </h1>
              </div>
              <div style={{ display: 'flex', gap: 12 }}>
                <button
                  onClick={() => {
                    const text = `Мой карьерный путь в направлении "${aiProfile.top_profession}" построен с помощью Talap AI! 🚀\nПройди тест и построй свой роадмап: ${window.location.origin}`;
                    navigator.clipboard.writeText(text);
                    alert(lang === 'ru' ? 'Ссылка скопирована в буфер обмена!' : 'Сілтеме буферге көшірілді!');
                  }}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px', borderRadius: 12,
                    background: '#FFFFFF', border: `1px solid ${C.hairline}`, cursor: 'pointer',
                    fontFamily: C.font, fontSize: 14, fontWeight: 700, color: C.ink700,
                    transition: 'all 200ms'
                  }}
                  onMouseEnter={e => e.currentTarget.style.background = C.blue50}
                  onMouseLeave={e => e.currentTarget.style.background = '#FFFFFF'}
                >
                  <Icon name="globe" size={16} color={C.blue} />
                  {lang === 'ru' ? 'Поделиться' : 'Бөлісу'}
                </button>
                <button
                  onClick={() => window.print()}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 8, padding: '10px 16px', borderRadius: 12,
                    background: C.blue, border: 'none', cursor: 'pointer',
                    fontFamily: C.font, fontSize: 14, fontWeight: 700, color: '#FFFFFF',
                    transition: 'all 200ms', boxShadow: '0 4px 12px rgba(20,72,255,0.2)'
                  }}
                  onMouseEnter={e => e.currentTarget.style.transform = 'translateY(-2px)'}
                  onMouseLeave={e => e.currentTarget.style.transform = 'translateY(0)'}
                >
                  <Icon name="compass" size={16} color="#FFFFFF" />
                  {lang === 'ru' ? 'Скачать PDF' : 'PDF жүктеу'}
                </button>
              </div>
            </div>



            {/* Print Only Header */}
            <div className="print-only" style={{ display: 'none', marginBottom: 30, borderBottom: `2px solid ${C.blue}`, paddingBottom: 20 }}>
              <h1 style={{ fontFamily: C.font, fontSize: 28, fontWeight: 800, color: C.ink900, marginBottom: 8 }}>
                Talap AI — Персональный карьерный путь
              </h1>
              <div style={{ fontFamily: C.font, fontSize: 16, color: C.ink700 }}>
                Рекомендуемая специальность: <strong>{aiProfile.top_profession}</strong>
              </div>
            </div>

            {/* Print-specific style tag */}
            <style>{`
              @media print {
                body {
                  background: #FFFFFF !important;
                  color: #000000 !important;
                }
                .no-print, header, nav, aside, .TopBar, button, .sidebar-container, .topbar-container {
                  display: none !important;
                }
                .print-only {
                  display: block !important;
                }
                main, .content-container {
                  margin: 0 !important;
                  padding: 0 !important;
                  width: 100% !important;
                }
                .card-container, div[style*="Card"] {
                  border: 1px solid #CCCCCC !important;
                  box-shadow: none !important;
                  page-break-inside: avoid;
                }
              }
            `}</style>

            <div style={{ 
              display: 'flex', 
              gap: 24, 
              overflowX: 'auto', 
              paddingBottom: 28,
              paddingTop: 12,
              scrollBehavior: 'smooth',
              WebkitOverflowScrolling: 'touch',
              scrollbarWidth: 'none', /* Firefox */
              msOverflowStyle: 'none',  /* IE and Edge */
            }} className="roadmap-scroll-container">
              
              <style>{`
                .roadmap-scroll-container::-webkit-scrollbar {
                  display: none; /* Chrome, Safari and Opera */
                }
              `}</style>
              
              {steps.map((step, i) => (
                <div 
                  key={i} 
                  style={{ 
                    flex: '0 0 320px', 
                    position: 'relative',
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'space-between'
                  }}
                >
                  {/* Step Circle & Connector */}
                  <div style={{ display: 'flex', alignItems: 'center', marginBottom: 20 }}>
                    <div style={{
                      width: 44, height: 44, borderRadius: 22, background: 'white', border: `2px solid ${C.blue}`,
                      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1, 
                      boxShadow: '0 4px 12px rgba(20,72,255,0.1)', flexShrink: 0
                    }}>
                      <span style={{ fontFamily: C.font, fontSize: 16, fontWeight: 800, color: C.blue }}>{i + 1}</span>
                    </div>
                    {i < steps.length - 1 && (
                      <div style={{ 
                        flex: 1, 
                        height: 2, 
                        background: C.blue100, 
                        marginLeft: 12, 
                        marginRight: -12,
                        position: 'relative'
                      }}>
                        {/* Arrowhead */}
                        <div style={{
                          position: 'absolute', right: 0, top: -4, width: 8, height: 8,
                          borderTop: `2px solid ${C.blue100}`, borderRight: `2px solid ${C.blue100}`,
                          transform: 'rotate(45deg)'
                        }} />
                      </div>
                    )}
                  </div>
                  
                  {/* Step Card */}
                  <Card style={{ padding: 24, flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'space-between', minHeight: 240 }}>
                    <div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 12, gap: 8 }}>
                        <h3 style={{ fontFamily: C.font, fontSize: 16, fontWeight: 700, color: C.ink900, margin: 0, lineHeight: 1.3 }}>{step.title}</h3>
                        {step.duration && (
                          <div style={{ padding: '3px 8px', borderRadius: 6, background: C.blue50, color: C.blue, fontFamily: '"Geist Mono",monospace', fontSize: 11, fontWeight: 700, flexShrink: 0 }}>
                            {step.duration}
                          </div>
                        )}
                      </div>
                      <p style={{ fontFamily: C.font, fontSize: 13, lineHeight: '20px', color: C.ink700, margin: 0 }}>
                        {step.description}
                      </p>
                    </div>
                    
                    {step.resources && (
                      <div style={{ marginTop: 16, display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                        {step.resources.map((res, j) => (
                          <div key={j} style={{ padding: '4px 8px', borderRadius: 6, background: C.mist, border: `1px solid ${C.hairline}`, display: 'flex', alignItems: 'center', gap: 6 }}>
                            <Icon name="compass" size={11} color={C.ink500} />
                            <span style={{ fontFamily: C.font, fontSize: 11, fontWeight: 600, color: C.ink700 }}>{res}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </Card>
                </div>
              ))}
            </div>

            {/* Large CTA card — get personalized roadmap */}
            <div className="no-print" style={{
              marginTop: 40,
              background: 'linear-gradient(135deg, #1448FF 0%, #3B6EFF 100%)',
              borderRadius: 24,
              padding: isMobile ? '28px 20px' : '36px 40px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              flexWrap: 'wrap',
              gap: 24,
              boxShadow: '0 20px 60px rgba(20,72,255,0.25)',
              position: 'relative',
              overflow: 'hidden'
            }}>
              {/* decorative circles */}
              <div style={{ position: 'absolute', right: -40, top: -40, width: 200, height: 200, borderRadius: '50%', background: 'rgba(255,255,255,0.06)', pointerEvents: 'none' }} />
              <div style={{ position: 'absolute', right: 60, bottom: -60, width: 140, height: 140, borderRadius: '50%', background: 'rgba(255,255,255,0.04)', pointerEvents: 'none' }} />

              <div style={{ flex: 1, minWidth: 260, position: 'relative' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
                  <div style={{ width: 32, height: 32, borderRadius: 8, background: 'rgba(255,255,255,0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <Icon name="sparkles" size={18} color="#FFFFFF" />
                  </div>
                  <span style={{ fontFamily: C.font, fontSize: 12, fontWeight: 700, color: 'rgba(255,255,255,0.7)', textTransform: 'uppercase', letterSpacing: '0.08em' }}>
                    {steps.length <= 5 ? 'Базовый план' : 'Уже создан'} · Talap AI
                  </span>
                </div>
                <h2 style={{ fontFamily: C.font, fontSize: isMobile ? 20 : 24, fontWeight: 800, color: '#FFFFFF', margin: '0 0 10px', letterSpacing: '-0.02em', lineHeight: 1.2 }}>
                  {steps.length <= 5 ? 'Получи детальный персональный роадмап' : 'Обнови свой роадмап под новые цели'}
                </h2>
                <p style={{ fontFamily: C.font, fontSize: 14, color: 'rgba(255,255,255,0.75)', margin: 0, lineHeight: 1.6, maxWidth: 480 }}>
                  {steps.length <= 5
                    ? 'Talap AI проведёт с тобой короткое интервью — спросит о твоих целях, навыках и сроках. После ответов сгенерирует детальный план с конкретными технологиями, ресурсами и дедлайнами.'
                    : 'Расскажи Talap AI о новых целях, и он обновит твой карьерный путь с учётом текущего прогресса.'}
                </p>
              </div>

              <button
                onClick={() => {
                  const prof = aiProfile?.top_profession || 'этой специальности';
                  const prompt = steps.length <= 5
                    ? `Привет! Я хочу получить детальный персональный карьерный роадмап по специальности «${prof}». У меня уже есть базовый план, но мне нужен индивидуальный с учётом моих целей, текущего уровня и временных рамок. Пожалуйста, задавай мне вопросы по одному — я отвечу, а потом ты составишь подробный пошаговый роадмап с конкретными технологиями, курсами и дедлайнами.`
                    : `Привет! Я хочу обновить свой карьерный роадмап по специальности «${prof}». Задай мне несколько вопросов о моём текущем прогрессе и новых целях, а затем сгенерируй обновлённый план.`;
                  localStorage.setItem('talap_chat_prefill', prompt);
                  navigate('/chat');
                }}
                style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '14px 24px', borderRadius: 14,
                  background: '#FFFFFF', border: 'none', cursor: 'pointer',
                  fontFamily: C.font, fontSize: 15, fontWeight: 800, color: '#1448FF',
                  boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
                  transition: 'all 200ms',
                  whiteSpace: 'nowrap',
                  flexShrink: 0,
                  position: 'relative'
                }}
                onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-3px)'; e.currentTarget.style.boxShadow = '0 12px 32px rgba(0,0,0,0.2)'; }}
                onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = '0 8px 24px rgba(0,0,0,0.15)'; }}
              >
                <Icon name="sparkles" size={18} color="#1448FF" />
                {steps.length <= 5 ? 'Получить детальный роадмап' : 'Обновить роадмап'}
                <Icon name="arrowRight" size={16} color="#1448FF" style={{ transform: 'rotate(-90deg)' }} />
              </button>
            </div>
          </>
        ) : (
          <div style={{ textAlign: 'center', padding: '60px 0', maxWidth: 480, margin: '0 auto' }}>
            <div style={{ width: 80, height: 80, borderRadius: 24, margin: '0 auto 24px', display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.blue50 }}>
              <Icon name="sparkles" size={40} color={C.blue} strokeWidth={1.5} />
            </div>
            <h2 style={{ fontFamily: C.font, fontSize: 24, fontWeight: 800, color: C.ink900, marginBottom: 12 }}>
              Построй подробный путь с Talap AI
            </h2>
            <p style={{ fontFamily: C.font, fontSize: 15, color: C.ink500, lineHeight: 1.6, marginBottom: 32 }}>
              Для получения детального горизонтального роадмапа необходимо пройти короткую ИИ-консультацию. Наш AI-советник проведет с тобой интервью прямо в чате и составит крутой пошаговый план развития.
            </p>
            <Button
              variant="primary"
              size="lg"
              onClick={() => {
                const prof = aiProfile?.top_profession || 'этой специальности';
                const prompt = `Привет! Я хочу получить детальный персональный карьерный роадмап по специальности «${prof}». Пожалуйста, задавай мне вопросы по одному — я отвечу на каждый, а потом ты составишь подробный пошаговый план с конкретными технологиями, курсами и дедлайнами. Начинай!`;
                localStorage.setItem('talap_chat_prefill', prompt);
                navigate('/chat');
              }}
              style={{ padding: '14px 28px' }}
            >
              Начать консультацию с Talap AI
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
