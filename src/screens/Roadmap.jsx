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
        {steps.length > 3 ? (
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
                const prof = aiProfile?.top_profession || 'своей будущей специальности';
                localStorage.setItem('talap_chat_prefill', `Привет! Я хочу составить подробный карьерный роадмап по направлению "${prof}". Проведи со мной интервью: задавай вопросы по очереди о моих целях и текущих навыках, а затем сгенерируй крутой роадмап.`);
                navigate('/chat');
              }}
              style={{ padding: '14px 28px' }}
            >
              Пройти ИИ-консультацию
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
