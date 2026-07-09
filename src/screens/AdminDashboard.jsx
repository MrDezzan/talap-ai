import { useState, useEffect } from 'react';
import { useMobile } from '../hooks/useMobile';
import Icon from '../components/Icon';
import Card from '../components/Card';
import Chip from '../components/Chip';
import TopBar from '../components/TopBar';
import { api } from '../lib/api';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';

const C = {
  ink900: '#0A1230', ink700: '#2A3457', ink500: '#5A6485', ink300: '#9AA3BF',
  blue: '#1448FF', blue50: '#F2F5FF', blue100: '#E6ECFF',
  paper: '#FFFFFF', mist: '#F5F7FB', hairline: '#E4E8F1',
  font: 'var(--font-sans)',
};

export default function AdminDashboard() {
  const isMobile = useMobile();
  const { lang, t } = useLanguage();
  const { fetchAIProfile } = useAuth();
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const data = await api.get('/admin/stats');
        setStats(data);
      } catch (err) {
        setError(err.error || err.message || 'Ошибка загрузки статистики');
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
  }, []);

  if (loading) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.mist, fontFamily: C.font }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{ width: 48, height: 48, borderRadius: '50%', border: `3px solid ${C.blue100}`, borderTopColor: C.blue, animation: 'spin 1s linear infinite', margin: '0 auto 16px' }} />
          <div style={{ color: C.ink500, fontSize: 14, fontWeight: 600 }}>{t('admin_loading')}</div>
        </div>
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.mist, padding: 24 }}>
        <Card style={{ maxWidth: 400, textAlign: 'center', padding: 32 }}>
          <div style={{ width: 56, height: 56, borderRadius: '50%', background: '#FFF0F0', color: '#FF4D4D', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 20px' }}>
            <Icon name="settings" size={24} color="#FF4D4D" />
          </div>
          <h3 style={{ fontFamily: C.font, fontSize: 18, fontWeight: 800, color: C.ink900, marginBottom: 8 }}>{t('admin_access_restricted')}</h3>
          <p style={{ fontFamily: C.font, fontSize: 14, color: C.ink500, lineHeight: 1.5, marginBottom: 24 }}>{error}</p>
        </Card>
      </div>
    );
  }

  const filteredUsers = stats?.users?.filter(u => 
    u.name?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    u.email?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    u.city?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    u.top_profession?.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];

  return (
    <div style={{ flex: 1, overflow: 'auto', background: C.mist, display: 'flex', flexDirection: 'column' }}>
      <TopBar title={t('admin_title')} />

      <div style={{ padding: isMobile ? 16 : 40, maxWidth: 1360, width: '100%', margin: '0 auto', boxSizing: 'border-box', display: 'flex', flexDirection: 'column', gap: 24 }}>
        
        {/* Row 1: KPI Stats Cards */}
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : 'repeat(5, 1fr)', gap: 16 }}>
          {[
            { label: t('admin_total_users'), value: stats.total_users, icon: 'user', color: '#1448FF', bg: '#F2F5FF' },
            { label: t('admin_grant_base'), value: stats.total_grants, icon: 'award', color: '#10B981', bg: '#E6FDF5' },
            { label: t('admin_catalog_profs'), value: stats.total_professions, icon: 'compass', color: '#F59E0B', bg: '#FFFBEB' },
            { label: t('admin_ai_chats'), value: stats.total_chats, icon: 'sparkles', color: '#8B5CF6', bg: '#F5F3FF' },
            { label: t('admin_total_msgs'), value: stats.total_messages, icon: 'target', color: '#EC4899', bg: '#FDF2F8' },
          ].map((card, i) => (
            <Card key={i} style={{ padding: 20, display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 44, height: 44, borderRadius: 12, background: card.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                <Icon name={card.icon} size={20} color={card.color} />
              </div>
              <div>
                <div style={{ fontFamily: C.font, fontSize: 12, fontWeight: 600, color: C.ink500, marginBottom: 4 }}>{card.label}</div>
                <div style={{ fontFamily: C.font, fontSize: 24, fontWeight: 800, color: C.ink900, lineHeight: 1 }}>{card.value}</div>
              </div>
            </Card>
          ))}
        </div>

        {/* Row 2: Breakdown distributions */}
        <div style={{ display: 'grid', gridTemplateColumns: isMobile ? '1fr' : '1fr 1fr', gap: 20 }}>
          
          {/* Cities breakdown */}
          <Card style={{ padding: 24 }}>
            <h3 style={{ fontFamily: C.font, fontSize: 15, fontWeight: 700, color: C.ink900, marginBottom: 16 }}>{t('admin_geography')}</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {Object.entries(stats.city_breakdown || {})
                .sort((a, b) => b[1] - a[1])
                .map(([city, count]) => {
                  const pct = Math.round((count / (stats.total_users || 1)) * 100);
                  const displayCity = city === 'Не указан' ? (lang === 'kz' ? 'Көрсетілмеген' : lang === 'en' ? 'Not specified' : 'Не указан') : city;
                  return (
                    <div key={city}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontFamily: C.font, fontSize: 13, fontWeight: 600, color: C.ink700, marginBottom: 6 }}>
                        <span>{displayCity}</span>
                        <span>{count} ({pct}%)</span>
                      </div>
                      <div style={{ height: 6, borderRadius: 3, background: C.blue50, overflow: 'hidden' }}>
                        <div style={{ height: '100%', background: C.blue, borderRadius: 3, width: `${pct}%`, transition: 'width 300ms ease' }} />
                      </div>
                    </div>
                  );
                })}
            </div>
          </Card>

          {/* Grades breakdown */}
          <Card style={{ padding: 24 }}>
            <h3 style={{ fontFamily: C.font, fontSize: 15, fontWeight: 700, color: C.ink900, marginBottom: 16 }}>{t('admin_grade_dist')}</h3>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {Object.entries(stats.grade_breakdown || {})
                .sort((a, b) => b[1] - a[1])
                .map(([grade, count]) => {
                  const pct = Math.round((count / (stats.total_users || 1)) * 100);
                  const displayGrade = grade === 'Не указан' 
                    ? (lang === 'kz' ? 'Көрсетілмеген' : lang === 'en' ? 'Not specified' : 'Не указан') 
                    : `${grade} ${lang === 'kz' ? 'сынып' : lang === 'en' ? 'grade' : 'класс'}`;
                  return (
                    <div key={grade}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontFamily: C.font, fontSize: 13, fontWeight: 600, color: C.ink700, marginBottom: 6 }}>
                        <span>{displayGrade}</span>
                        <span>{count} ({pct}%)</span>
                      </div>
                      <div style={{ height: 6, borderRadius: 3, background: C.blue50, overflow: 'hidden' }}>
                        <div style={{ height: '100%', background: '#10B981', borderRadius: 3, width: `${pct}%`, transition: 'width 300ms ease' }} />
                      </div>
                    </div>
                  );
                })}
            </div>
          </Card>
        </div>

        {/* Row 3: Users detailed table */}
        <Card style={{ padding: 24, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20, flexWrap: 'wrap', gap: 12 }}>
            <div>
              <h3 style={{ fontFamily: C.font, fontSize: 16, fontWeight: 800, color: C.ink900, margin: 0 }}>{t('admin_user_list')}</h3>
              <div style={{ fontFamily: C.font, fontSize: 12, color: C.ink500, marginTop: 4 }}>{t('admin_found_users')}: {filteredUsers.length}</div>
            </div>
            
            {/* Search Input */}
            <div style={{
              display: 'flex', alignItems: 'center', gap: 8, padding: '8px 14px', borderRadius: 10,
              background: C.mist, border: `1px solid ${C.hairline}`, width: isMobile ? '100%' : 260
            }}>
              <Icon name="search" size={14} color={C.ink500} />
              <input
                type="text"
                placeholder={t('admin_search_placeholder')}
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                style={{ background: 'none', border: 'none', outline: 'none', fontSize: 13, fontFamily: C.font, width: '100%', color: C.ink900 }}
              />
            </div>
          </div>

          <div style={{ overflowX: 'auto', width: '100%' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: `2px solid ${C.hairline}` }}>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>ID</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{lang === 'kz' ? 'Пайдаланушы' : lang === 'en' ? 'User' : 'Пользователь'}</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{t('admin_role')}</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{t('admin_grade_school')}</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{t('admin_city')}</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{t('admin_ai_recommendation')}</th>
                  <th style={{ padding: '12px 16px', fontFamily: C.font, fontSize: 12, fontWeight: 700, color: C.ink500 }}>{t('admin_registration')}</th>
                </tr>
              </thead>
              <tbody>
                {filteredUsers.map(u => (
                  <tr key={u.id} style={{ borderBottom: `1px solid ${C.hairline}`, transition: 'background 150ms' }} className="user-row">
                    <td style={{ padding: '14px 16px', fontFamily: '"Geist Mono",monospace', fontSize: 12, color: C.ink500 }}>#{u.id}</td>
                    <td style={{ padding: '14px 16px' }}>
                      <div style={{ fontFamily: C.font, fontSize: 14, fontWeight: 700, color: C.ink900 }}>{u.name}</div>
                      <div style={{ fontFamily: C.font, fontSize: 11, color: C.ink500, marginTop: 2 }}>{u.email}</div>
                    </td>
                    <td style={{ padding: '14px 16px' }}>
                      <Chip tone={u.role === 'admin' ? 'ai' : 'default'} style={{ fontSize: 10 }}>
                        {u.role === 'admin' ? t('admin_role_admin') : t('admin_student')}
                      </Chip>
                    </td>
                    <td style={{ padding: '14px 16px' }}>
                      <div style={{ fontFamily: C.font, fontSize: 13, fontWeight: 600, color: C.ink700 }}>{u.grade ? `${u.grade} ${lang === 'kz' ? 'сынып' : lang === 'en' ? 'grade' : 'класс'}` : '—'}</div>
                      <div style={{ fontFamily: C.font, fontSize: 11, color: C.ink500, marginTop: 2, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 180 }}>{u.school || '—'}</div>
                    </td>
                    <td style={{ padding: '14px 16px', fontFamily: C.font, fontSize: 13, fontWeight: 600, color: C.ink700 }}>
                      {u.city === 'Не указан' || !u.city ? '—' : u.city}
                    </td>
                    <td style={{ padding: '14px 16px' }}>
                      <Chip tone={u.top_profession === 'Не пройден' ? 'warn' : 'blue'} style={{ fontSize: 11, fontWeight: 700 }}>
                        {u.top_profession === 'Не пройден' ? t('admin_not_passed') : u.top_profession}
                      </Chip>
                    </td>
                    <td style={{ padding: '14px 16px', fontFamily: C.font, fontSize: 12, color: C.ink500 }}>
                      {new Date(u.created_at).toLocaleDateString(lang === 'kz' ? 'kk-KZ' : lang === 'en' ? 'en-US' : 'ru-RU', { day: 'numeric', month: 'short', year: 'numeric' })}
                    </td>
                  </tr>
                ))}
                {filteredUsers.length === 0 && (
                  <tr>
                    <td colSpan="7" style={{ padding: '40px 16px', textAlign: 'center', fontFamily: C.font, fontSize: 14, color: C.ink300 }}>
                      {t('admin_users_not_found')}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
      <style>{`
        .user-row:hover {
          background: #F9FAFC;
        }
      `}</style>
    </div>
  );
}
