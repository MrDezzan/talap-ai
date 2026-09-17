import { useState, useEffect } from 'react';
import Icon from './Icon';
import Card from './Card';
import Chip from './Chip';
import { api } from '../lib/api';

const C = {
  ink900: '#0A1230', ink700: '#2A3457', ink500: '#5A6485', ink300: '#9AA3BF',
  blue: '#1448FF', blue50: '#F2F5FF', blue100: '#E6ECFF',
  paper: '#FFFFFF', mist: '#F5F7FB', hairline: '#E4E8F1',
  font: 'var(--font-sans)',
};

const inputStyle = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '10px 12px',
  borderRadius: 8,
  border: `1px solid ${C.hairline}`,
  fontFamily: C.font,
  fontSize: 13,
  color: C.ink900,
  outline: 'none',
  background: C.paper,
};

const labelStyle = {
  fontFamily: C.font,
  fontSize: 12,
  fontWeight: 600,
  color: C.ink500,
  marginBottom: 6,
  display: 'block',
};

/**
 * Generic admin CRUD section (list + create/edit form overlay + delete)
 * used for both Grants and Professions management in AdminDashboard.
 *
 * Props:
 * - t, lang: from useLanguage()
 * - title: string (section heading)
 * - addLabel: string (button label for "add new")
 * - listPath: public GET endpoint returning the full published list, e.g. '/grants'
 * - adminBasePath: admin CRUD base path, e.g. '/admin/grants'
 * - fields: [{ key, label, type: 'text'|'textarea'|'number'|'tags'|'checkbox', placeholder? }]
 * - renderRowMeta: (item) => node — small subtitle/status rendered under the name
 * - emptyText, loadingText: strings
 */
export default function AdminCrudSection({
  t, lang, title, addLabel, listPath, adminBasePath, fields, renderRowMeta,
}) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [editing, setEditing] = useState(null); // null = closed, {} = new, {...item} = edit
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState(null);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.get(listPath);
      setItems(Array.isArray(data) ? data : []);
    } catch (err) {
      setError(err.message || 'Error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const openCreate = () => {
    const draft = {};
    fields.forEach(f => {
      if (f.type === 'checkbox') draft[f.key] = false;
      else if (f.type === 'tags') draft[f.key] = '';
      else draft[f.key] = '';
    });
    setEditing(draft);
  };

  const openEdit = (item) => {
    const draft = { ...item };
    fields.forEach(f => {
      if (f.type === 'tags') {
        draft[f.key] = Array.isArray(item[f.key]) ? item[f.key].join(', ') : (item[f.key] || '');
      }
    });
    setEditing(draft);
  };

  const closeForm = () => setEditing(null);

  const handleChange = (key, value) => {
    setEditing(prev => ({ ...prev, [key]: value }));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = { ...editing };
      fields.forEach(f => {
        if (f.type === 'tags') {
          payload[f.key] = String(payload[f.key] || '')
            .split(',')
            .map(s => s.trim())
            .filter(Boolean);
        } else if (f.type === 'number') {
          payload[f.key] = payload[f.key] === '' || payload[f.key] === undefined ? undefined : Number(payload[f.key]);
        }
      });

      if (editing.id) {
        const updated = await api.put(`${adminBasePath}/${editing.id}`, payload);
        setItems(prev => prev.map(it => (it.id === editing.id ? { ...it, ...updated } : it)));
      } else {
        const created = await api.post(adminBasePath, payload);
        setItems(prev => [created, ...prev]);
      }
      setEditing(null);
    } catch (err) {
      alert(err.message || 'Ошибка сохранения');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item) => {
    if (!window.confirm(t('admin_confirm_delete'))) return;
    setDeletingId(item.id);
    try {
      await api.del(`${adminBasePath}/${item.id}`);
      setItems(prev => prev.filter(it => it.id !== item.id));
    } catch (err) {
      alert(err.message || 'Ошибка удаления');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <Card style={{ padding: 24 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 16, flexWrap: 'wrap', gap: 8 }}>
        <div>
          <h3 style={{ fontFamily: C.font, fontSize: 16, fontWeight: 800, color: C.ink900, margin: 0 }}>
            {title} {items.length > 0 && `(${items.length})`}
          </h3>
        </div>
        <button
          onClick={openCreate}
          style={{
            padding: '8px 16px', borderRadius: 8, border: 'none', background: C.blue, color: '#fff',
            fontFamily: C.font, fontSize: 12, fontWeight: 700, cursor: 'pointer',
          }}
        >
          {addLabel}
        </button>
      </div>

      {loading ? (
        <div style={{ fontFamily: C.font, fontSize: 13, color: C.ink500, padding: '12px 0' }}>{t('admin_pending_loading')}</div>
      ) : error ? (
        <div style={{ fontFamily: C.font, fontSize: 13, color: '#FF4D4D', padding: '12px 0' }}>{error}</div>
      ) : items.length === 0 ? (
        <div style={{ fontFamily: C.font, fontSize: 13, color: C.ink300, padding: '12px 0' }}>{t('admin_pending_empty')}</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {items.map(item => (
            <div key={item.id} style={{ border: `1px solid ${C.hairline}`, borderRadius: 12, padding: 16, display: 'flex', justifyContent: 'space-between', gap: 16, flexWrap: 'wrap' }}>
              <div style={{ flex: 1, minWidth: 220 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6, flexWrap: 'wrap' }}>
                  <span style={{ fontFamily: C.font, fontSize: 14, fontWeight: 700, color: C.ink900 }}>{item.name}</span>
                  {item.status && <Chip tone="default" style={{ fontSize: 10 }}>{item.status}</Chip>}
                  {item.hot && <Chip tone="warn" style={{ fontSize: 10 }}>🔥</Chip>}
                </div>
                {renderRowMeta ? renderRowMeta(item) : null}
              </div>
              <div style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
                <button
                  onClick={() => openEdit(item)}
                  style={{
                    padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.hairline}`, background: '#fff', color: C.ink700,
                    fontFamily: C.font, fontSize: 12, fontWeight: 700, cursor: 'pointer',
                  }}
                >
                  {t('admin_edit')}
                </button>
                <button
                  onClick={() => handleDelete(item)}
                  disabled={deletingId === item.id}
                  style={{
                    padding: '8px 16px', borderRadius: 8, border: `1px solid ${C.hairline}`, background: '#fff', color: '#FF4D4D',
                    fontFamily: C.font, fontSize: 12, fontWeight: 700, cursor: 'pointer', opacity: deletingId === item.id ? 0.6 : 1,
                  }}
                >
                  {t('admin_delete')}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <div
          onClick={closeForm}
          style={{
            position: 'fixed', inset: 0, background: 'rgba(10,18,48,0.5)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            zIndex: 300, backdropFilter: 'blur(4px)', padding: 24,
          }}
        >
          <div
            onClick={e => e.stopPropagation()}
            style={{
              background: C.paper, borderRadius: 16, width: '100%', maxWidth: 560,
              maxHeight: '85vh', overflow: 'auto', boxShadow: '0 32px 80px rgba(10,18,48,0.2)',
              padding: 28,
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
              <h3 style={{ fontFamily: C.font, fontSize: 18, fontWeight: 800, color: C.ink900, margin: 0 }}>
                {editing.id ? t('admin_edit') : addLabel}
              </h3>
              <button
                onClick={closeForm}
                style={{ width: 32, height: 32, borderRadius: 8, background: C.mist, border: 'none', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}
              >
                <Icon name="plus" size={16} color={C.ink500} style={{ transform: 'rotate(45deg)' }} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {fields.map(f => (
                <div key={f.key}>
                  {f.type === 'checkbox' ? (
                    <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={!!editing[f.key]}
                        onChange={e => handleChange(f.key, e.target.checked)}
                      />
                      <span style={{ ...labelStyle, marginBottom: 0 }}>{f.label}</span>
                    </label>
                  ) : (
                    <>
                      <label style={labelStyle}>{f.label}</label>
                      {f.type === 'textarea' ? (
                        <textarea
                          value={editing[f.key] ?? ''}
                          onChange={e => handleChange(f.key, e.target.value)}
                          rows={3}
                          style={{ ...inputStyle, resize: 'vertical', fontFamily: C.font }}
                        />
                      ) : (
                        <input
                          type={f.type === 'number' ? 'number' : 'text'}
                          value={editing[f.key] ?? ''}
                          onChange={e => handleChange(f.key, e.target.value)}
                          placeholder={f.placeholder}
                          style={inputStyle}
                        />
                      )}
                    </>
                  )}
                </div>
              ))}
            </div>

            <div style={{ display: 'flex', gap: 10, marginTop: 24 }}>
              <button
                onClick={handleSave}
                disabled={saving}
                style={{
                  flex: 1, padding: '11px 18px', borderRadius: 10, border: 'none', background: C.blue, color: '#fff',
                  fontFamily: C.font, fontSize: 14, fontWeight: 700, cursor: 'pointer', opacity: saving ? 0.6 : 1,
                }}
              >
                {t('admin_save')}
              </button>
              <button
                onClick={closeForm}
                style={{
                  flex: 1, padding: '11px 18px', borderRadius: 10, border: `1px solid ${C.hairline}`, background: '#fff', color: C.ink700,
                  fontFamily: C.font, fontSize: 14, fontWeight: 700, cursor: 'pointer',
                }}
              >
                {t('admin_cancel')}
              </button>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}
