import { useState } from 'react';
import { NavLink, Link, useNavigate } from 'react-router-dom';
import Icon from './Icon';
import Avatar from './Avatar';
import Button from './Button';
import logomark from '../assets/logomark.svg';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { useMobile } from '../hooks/useMobile';

export default function Sidebar() {
  const { user, logout } = useAuth();
  const { lang, setLang, t } = useLanguage();
  const isMobile = useMobile();
  const navigate = useNavigate();
  const [showMenu, setShowMenu] = useState(false);
  
  const navItems = [
    { to: '/dashboard',   label: t('nav_home'),          icon: 'home' },
    { to: '/professions', label: t('nav_professions'),   icon: 'compass' },
    { to: '/grants',      label: t('nav_grants'),        icon: 'award' },
    { to: '/roadmap',     label: t('nav_roadmap'),       icon: 'target' },
    { to: '/portfolio',   label: t('nav_portfolio'),     icon: 'trophy' },
    { to: '/chat',        label: t('nav_chat'),          icon: 'sparkles' },
  ];

  if (isMobile) {
    const mobileNavItems = [
      { to: '/dashboard',   label: t('nav_home'),          icon: 'home' },
      { to: '/professions', label: t('nav_professions'),   icon: 'compass' },
      { to: '/chat',        label: t('nav_chat'),          icon: 'sparkles', highlight: true },
      { to: '/grants',      label: t('nav_grants'),        icon: 'award' },
    ];

    return (
      <>
        <nav style={{
          position: 'fixed', bottom: 0, left: 0, right: 0,
          height: 64, background: '#FFFFFF', borderTop: '1px solid #E4E8F1',
          display: 'flex', justifyContent: 'space-around', alignItems: 'center',
          padding: '0 4px', zIndex: 1000, boxShadow: '0 -2px 12px rgba(10,18,48,0.06)'
        }}>
          {mobileNavItems.map(item => (
            <NavLink
              key={item.to}
              to={item.to}
              style={({ isActive }) => ({
                display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4,
                textDecoration: 'none', color: isActive ? '#1448FF' : '#5A6485',
                fontFamily: 'var(--font-sans)', fontSize: 10, fontWeight: isActive ? 700 : 500,
                flex: 1, padding: '6px 0', transition: 'color 150ms ease'
              })}
            >
              {({ isActive }) => (
                <>
                  {item.highlight ? (
                    <div style={{
                      width: 44, height: 44, borderRadius: 14,
                      background: isActive ? 'linear-gradient(135deg, #1448FF 0%, #3B6EFF 100%)' : '#F2F5FF',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      boxShadow: isActive ? '0 4px 12px rgba(20,72,255,0.3)' : 'none',
                      marginTop: -20, transition: 'all 200ms ease',
                      border: '3px solid #FFFFFF'
                    }}>
                      <Icon name={item.icon} size={20} color={isActive ? '#FFFFFF' : '#1448FF'} strokeWidth={2.25} />
                    </div>
                  ) : (
                    <Icon name={item.icon} size={20} strokeWidth={isActive ? 2 : 1.75} />
                  )}
                  <span style={{ fontSize: 9.5, marginTop: item.highlight ? 2 : 0 }}>{item.label}</span>
                </>
              )}
            </NavLink>
          ))}
          <button
            onClick={() => setShowMenu(true)}
            style={{
              display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4,
              background: 'none', border: 'none', color: '#5A6485', cursor: 'pointer',
              fontFamily: 'var(--font-sans)', fontSize: 9.5, fontWeight: 500, flex: 1, padding: '6px 0'
            }}
          >
            <Avatar name={user?.name || 'U'} size={20} />
            <span>Меню</span>
          </button>
        </nav>

        {/* Sliding Bottom Sheet Menu for Mobile */}
        {showMenu && (
          <div style={{
            position: 'fixed', inset: 0, zIndex: 10001,
            display: 'flex', flexDirection: 'column', justifyContent: 'flex-end',
            pointerEvents: 'auto'
          }}>
            {/* Backdrop with Blur */}
            <div 
              onClick={() => setShowMenu(false)}
              style={{
                position: 'absolute', inset: 0,
                background: 'rgba(10, 18, 48, 0.4)',
                backdropFilter: 'blur(4px)',
                animation: 'fadeIn 200ms ease'
              }} 
            />
            {/* Sliding Panel */}
            <div style={{
              position: 'relative',
              background: '#FFFFFF',
              borderTopLeftRadius: 24, borderTopRightRadius: 24,
              padding: '16px 20px 30px',
              boxShadow: '0 -10px 40px rgba(10,18,48,0.12)',
              zIndex: 10002,
              display: 'flex', flexDirection: 'column', gap: 18,
              animation: 'slideUp 250ms cubic-bezier(0.16, 1, 0.3, 1)'
            }}>
              {/* Drag Handle indicator */}
              <div style={{
                width: 36, height: 4, borderRadius: 2,
                background: '#E4E8F1', margin: '0 auto 6px'
              }} />
              
              {/* Header profile info */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, borderBottom: '1px solid #E4E8F1', paddingBottom: 16 }}>
                <Avatar name={user?.name || 'U'} size={44} />
                <div style={{ minWidth: 0, flex: 1 }}>
                  <div style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 700, color: '#0A1230', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {user?.name || 'Студент'}
                  </div>
                  <div style={{ fontFamily: 'var(--font-sans)', fontSize: 12, color: '#5A6485', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {user?.email || ''}
                  </div>
                </div>
              </div>

              {/* Navigation lists */}
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {[
                  { to: '/roadmap', label: t('nav_roadmap'), icon: 'target' },
                  { to: '/portfolio', label: t('nav_portfolio'), icon: 'trophy' },
                  { to: '/settings', label: t('nav_settings'), icon: 'settings' },
                ].map(item => (
                  <button
                    key={item.to}
                    onClick={() => {
                      setShowMenu(false);
                      navigate(item.to);
                    }}
                    style={{
                      display: 'flex', alignItems: 'center', gap: 12,
                      padding: '12px 16px', borderRadius: 12,
                      background: '#F5F7FB', border: 'none', cursor: 'pointer',
                      fontFamily: 'var(--font-sans)', fontSize: 14, fontWeight: 600,
                      color: '#2A3457', textAlign: 'left', width: '100%'
                    }}
                  >
                    <Icon name={item.icon} size={18} color="#1448FF" />
                    {item.label}
                  </button>
                ))}
              </div>

              {/* Language Selection */}
              <div style={{ display: 'flex', gap: 6, borderTop: '1px solid #E4E8F1', paddingTop: 16 }}>
                {['ru', 'en', 'kz'].map(l => (
                  <button
                    key={l}
                    onClick={() => setLang(l)}
                    style={{
                      flex: 1, padding: '8px 0', borderRadius: 8,
                      border: '1px solid ' + (lang === l ? '#1448FF' : '#E4E8F1'),
                      background: lang === l ? '#F2F5FF' : 'white',
                      color: lang === l ? '#1448FF' : '#5A6485',
                      fontSize: 12, fontWeight: 700, textTransform: 'uppercase',
                      cursor: 'pointer', transition: 'all 150ms'
                    }}
                  >
                    {l}
                  </button>
                ))}
              </div>

              {/* Logout Button */}
              <button
                onClick={() => {
                  setShowMenu(false);
                  logout();
                }}
                style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
                  padding: '12px 0', borderRadius: 12,
                  background: '#FFF0F0', border: 'none', cursor: 'pointer',
                  fontFamily: 'var(--font-sans)', fontSize: 13, fontWeight: 700,
                  color: '#FF4D4D', width: '100%'
                }}
              >
                Выйти из аккаунта
              </button>
            </div>
          </div>
        )}

        {/* Global animations style tag */}
        <style>{`
          @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
          }
          @keyframes slideUp {
            from { transform: translateY(100%); }
            to { transform: translateY(0); }
          }
        `}</style>
      </>
    );
  }


  return (
    <aside style={{
      width: 240,
      height: '100%',
      background: '#FFFFFF',
      borderRight: '1px solid #E4E8F1',
      display: 'flex',
      flexDirection: 'column',
      flexShrink: 0,
    }}>
      <Link to="/dashboard" style={{ padding: '20px 16px 24px', display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none', cursor: 'pointer' }}>
        <img src={logomark} alt="Talap Logo" style={{ width: 32, height: 32, borderRadius: 8 }} />
        <div style={{ fontFamily: 'var(--font-sans)', fontSize: 18, fontWeight: 800, letterSpacing: '-0.03em', color: '#0A1230' }}>
          Talap<span style={{ color: '#1448FF' }}>.</span>ai
        </div>
      </Link>

      <nav style={{ padding: '0 8px', display: 'flex', flexDirection: 'column', gap: 2 }}>
        {navItems.map(item => (
          <NavLink
            key={item.to}
            to={item.to}
            style={({ isActive }) => ({
              display: 'flex',
              alignItems: 'center',
              gap: 12,
              padding: '10px 12px',
              background: isActive ? '#F2F5FF' : 'transparent',
              border: 'none',
              borderRadius: 8,
              cursor: 'pointer',
              textAlign: 'left',
              fontFamily: 'var(--font-sans)',
              fontSize: 14,
              fontWeight: isActive ? 600 : 500,
              color: isActive ? '#1448FF' : '#2A3457',
              textDecoration: 'none',
              transition: 'background 150ms',
            })}
          >
            {({ isActive }) => (
              <>
                <Icon name={item.icon} size={18} strokeWidth={isActive ? 2 : 1.75} />
                {item.label}
              </>
            )}
          </NavLink>
        ))}
      </nav>

      <div style={{ flex: 1 }} />

      <div style={{ padding: '8px 16px', display: 'flex', gap: 4, marginBottom: 8 }}>
        {['ru', 'en', 'kz'].map(l => (
          <button
            key={l}
            onClick={() => setLang(l)}
            style={{
              flex: 1,
              padding: '6px 0',
              borderRadius: 6,
              border: '1px solid ' + (lang === l ? '#1448FF' : '#E4E8F1'),
              background: lang === l ? '#F2F5FF' : 'white',
              color: lang === l ? '#1448FF' : '#5A6485',
              fontSize: 11,
              fontWeight: 700,
              textTransform: 'uppercase',
              cursor: 'pointer',
              transition: 'all 150ms'
            }}
          >
            {l}
          </button>
        ))}
      </div>

      <Link to="/settings" style={{ padding: '12px 16px', borderTop: '1px solid #E4E8F1', display: 'flex', alignItems: 'center', gap: 10, textDecoration: 'none' }}>
        <Avatar name={user?.name || 'U'} size={32} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontFamily: 'var(--font-sans)', fontSize: 13, fontWeight: 600, color: '#0A1230', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{user?.name || t('nav_settings')}</div>
          <div style={{ fontFamily: 'var(--font-sans)', fontSize: 11, color: '#5A6485', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{user?.email || ''}</div>
        </div>
        <Icon name="settings" size={16} color="#5A6485" />
      </Link>
    </aside>
  );
}
